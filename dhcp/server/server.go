package server

import (
	"context"
	"dhcp/pool"
	"dhcp/protocol"
	"dhcp/transport"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	SELECTING = iota
	INIT_REBOOT
	RENEWING
	REBINDING
	InvalidState         = -1
	defaultMTU           = 1500
	defaultReadTimeout   = 500 * time.Millisecond
	leaseCleanupInterval = 1 * time.Minute
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1500)
	},
}

type Server struct {
	mu          sync.RWMutex
	bindings    map[uint64]*binding
	allocated   map[uint32]bool
	ipPool      *pool.IPPool
	config      *Config
	conn        net.PacketConn
	wg          sync.WaitGroup
	processChan chan *input
	mtu         int

	cachedReplyOptions *protocol.ReplyOptions
}

type input struct {
	data []byte
	addr *net.UDPAddr
}

type Config struct {
	Start         net.IP
	End           net.IP
	Subnet        net.IPNet
	Lease         time.Duration
	RenewalTime   time.Duration
	RebindingTime time.Duration
	DNS           []net.IP
	Router        net.IP
	ServerIP      net.IP
	DomainName    string
}

func (c *Config) Validate() error {
	if c.Lease <= 0 {
		return errors.New("lease duration must be positive")
	}
	if !c.Subnet.Contains(c.ServerIP) {
		return errors.New("server IP must be within subnet")
	}
	return nil
}

type binding struct {
	IP         net.IP
	MAC        net.HardwareAddr
	Expiration time.Time
}

type Offer struct {
	ClientMAC net.HardwareAddr
	OfferIP   net.IP
	ServerIP  net.IP
}

func NewServer(cfg *Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ipPool, err := pool.NewIPPool(cfg.Start, cfg.End)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP pool: %w", err)
	}

	s := &Server{
		bindings:    make(map[uint64]*binding),
		allocated:   make(map[uint32]bool),
		ipPool:      ipPool,
		config:      cfg,
		processChan: make(chan *input, 100),
	}
	s.mtu, err = transport.GetMTU()
	if err != nil {
		slog.Error("Error getting MTU, using default", "error", err, "defaultMTU", defaultMTU)
		s.mtu = defaultMTU
	}

	conn, err := transport.BuildConn()
	if err != nil {
		return nil, fmt.Errorf("failed to build connection: %w", err)
	}
	s.conn = conn

	return s, nil
}

func (s *Server) Run() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	withCancel, cancelFunc := context.WithCancel(context.Background())
	runAsync(withCancel, &s.wg, s.run)

	select {
	case <-sig:
		slog.Info("Received signal, stopping server")
		cancelFunc()
		close(s.processChan)
	}
	slog.Info("waiting for all goroutines to finish")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All goroutines completed")
	case <-ctx.Done():
		slog.Error("Timed out waiting for goroutines to complete")
	}

	slog.Info("Server stopped")
}

func (s *Server) run(ctx context.Context) {
	runAsync(ctx, &s.wg, s.processPackets)
	runAsync(ctx, &s.wg, s.cleanupExpiredLeases)
	runAsync(ctx, &s.wg, s.startReadConn)
}

func runAsync(ctx context.Context, wg *sync.WaitGroup, f func(ctx context.Context)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f(ctx)
	}()
}

func (s *Server) startReadConn(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = s.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))
			buf := bufPool.Get().([]byte)
			n, addr, err := s.conn.ReadFrom(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				slog.Error("error reading packet:", "error", err)
			}

			upeer, ok := addr.(*net.UDPAddr)
			if !ok {
				slog.Error("Invalid UDP address", "addr", addr)
				continue
			}

			s.processChan <- &input{data: buf[:n], addr: upeer}
			bufPool.Put(buf)
		}
	}
}

func (s *Server) processPackets(ctx context.Context) {
	for i := range s.processChan {
		packet, err := protocol.Decode(i.data)
		if err != nil {
			slog.Error("Error decoding packet", "error", err)
			continue
		}
		runAsync(ctx, &s.wg, func(ctx context.Context) {
			slog.Info("Processing packet", "packet", packet, "addr", i.addr)
			s.handlePacket(packet, i.addr)
		})
	}
}

func (s *Server) handlePacket(packet *protocol.Packet, addr *net.UDPAddr) {
	slog.Info("Received packet", "packet", packet, "addr", addr)
	switch packet.DHCPMessageType() {
	case protocol.DHCPDISCOVER:
		s.handleDiscover(packet, addr)
	case protocol.DHCPREQUEST:
		s.handleRequest(packet, addr)
	case protocol.DHCPRELEASE:
		s.handleRelease(packet)
	case protocol.DHCPDECLINE:
		s.handleDecline(packet)
	}
}

func (s *Server) handleDiscover(packet *protocol.Packet, addr *net.UDPAddr) {
	offer := s.createOffer(packet)
	if offer == nil {
		slog.Debug("No IP available for offer")
		return
	}
	err := protocol.SendPacket(s.conn, offer, addr)
	if err != nil {
		s.releaseIP(offer.YIAddr)
		slog.Error("Error sending offer", "error", err)
	}
}

func (s *Server) createOffer(packet *protocol.Packet) *protocol.Packet {
	ip := s.ipPool.Allocate()
	if ip == nil {
		return nil
	}

	slog.Info("Allocated IP", "ip", ip)
	offer := packet.ToOffer(ip, s.createReplyOptions())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings[MACToUint64(packet.CHAddr)] = &binding{
		IP:         ip,
		MAC:        packet.CHAddr,
		Expiration: time.Now().Add(s.config.Lease),
	}
	s.allocated[IPToUint32(ip)] = true
	slog.Info("Offering IP", "app", ip, "addr", packet.CHAddr.String())
	return offer
}

func (s *Server) handleRelease(packet *protocol.Packet) {
	s.releaseIP(packet.CIAddr)
}

func (s *Server) handleDecline(packet *protocol.Packet) {
	s.releaseIP(packet.CIAddr)
}

func (s *Server) releaseIP(ip net.IP) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ipUint := IPToUint32(ip)
	if _, exists := s.allocated[ipUint]; exists {
		delete(s.allocated, ipUint)
		s.ipPool.Release(ip)
	}

	for mac, b := range s.bindings {
		if b.IP.Equal(ip) {
			delete(s.bindings, mac)
			break
		}
	}
}

func (s *Server) setupListener() (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return nil, err
	}
	slog.Info("Listening on", "addr", conn.LocalAddr())
	return conn, nil
}

func (s *Server) cleanupExpiredLeases(ctx context.Context) {
	ticker := time.NewTicker(leaseCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for mac, b := range s.bindings {
				if b.Expiration.Before(now) {
					s.releaseIP(b.IP)
					delete(s.bindings, mac)
				}
			}
			s.mu.Unlock()
		case <-ctx.Done():
			slog.Info("Stopping lease cleanup")
			return
		}
	}
}

func (s *Server) createReplyOptions() *protocol.ReplyOptions {
	s.mu.RLock()
	if s.cachedReplyOptions != nil {
		defer s.mu.RUnlock()
		return s.cachedReplyOptions
	}
	s.mu.RUnlock()

	options := &protocol.ReplyOptions{
		LeaseTime:     s.config.Lease,
		RenewalTime:   s.config.RenewalTime,
		RebindingTime: s.config.RebindingTime,
		SubnetMask:    s.config.Subnet.Mask,
		Router:        s.config.Router,
		DNS:           s.config.DNS,
		ServerIP:      s.config.ServerIP,
		DomainName:    s.config.DomainName,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedReplyOptions = options
	return options
}

func (s *Server) createAckOrNak(packet *protocol.Packet) *protocol.Packet {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.bindings[MACToUint64(packet.CHAddr)]
	if !exists || !b.IP.Equal(packet.CIAddr) {
		slog.Error("Invalid request", "packet", packet)
		return packet.ToNak(s.createReplyOptions())
	}

	b.Expiration = time.Now().Add(s.config.Lease)
	slog.Info("Acknowledging IP", "ip", b.IP)
	return packet.ToAck(b.IP, s.createReplyOptions())
}

func (s *Server) handleRequest(packet *protocol.Packet, addr *net.UDPAddr) {
	state := determineClientState(packet)
	var response *protocol.Packet
	switch state {
	case SELECTING:
		s.mu.Lock()
		defer s.mu.Unlock()

		requestedIP := packet.GetOption(protocol.OptionRequestedIPAddress)
		serverIdentifier := packet.GetOption(protocol.OptionServerIdentifier)

		if !net.IP(serverIdentifier).Equal(s.config.ServerIP) {
			// Client has selected a different server
			return
		}
		response = s.buildResponseToBinding(packet, requestedIP)

	case INIT_REBOOT:
		s.mu.Lock()
		defer s.mu.Unlock()
		requestedIP := packet.GetOption(protocol.OptionRequestedIPAddress)
		response = s.buildResponseToBinding(packet, requestedIP)

	case RENEWING, REBINDING:
		s.mu.Lock()
		defer s.mu.Unlock()
		response = s.buildResponseToBinding(packet, packet.CIAddr)

	default:
		slog.Error("Invalid DHCPREQUEST state")
		return
	}
	if response == nil {
		slog.Error("Error creating response")
		return
	}
	err := protocol.SendPacket(s.conn, response, addr)
	if err != nil {
		slog.Error("Error sending response", "error", err)
	}
}

func (s *Server) buildResponseToBinding(packet *protocol.Packet, ip net.IP) (response *protocol.Packet) {
	b, exists := s.bindings[MACToUint64(packet.CHAddr)]
	isWrongBind := !exists || !b.IP.Equal(ip)
	expiredBind := b.Expiration.Before(time.Now())

	switch {
	case isWrongBind:
		return packet.ToNak(s.createReplyOptions())
	case expiredBind:
		return packet.ToNak(s.createReplyOptions())
	default:
		b.Expiration = time.Now().Add(s.config.Lease)
		return packet.ToAck(b.IP, s.createReplyOptions())
	}
}

func isZeroIP(ip net.IP) bool {
	return ip == nil || ip.Equal(net.IPv4zero)
}

func determineClientState(packet *protocol.Packet) int {
	if packet == nil {
		return InvalidState
	}

	emptyServer := isZeroIP(packet.SIAddr)
	hasRequestedIP := packet.GetOption(protocol.OptionRequestedIPAddress) != nil
	clientIPZero := isZeroIP(packet.CIAddr)
	isBroadcast := packet.IsBroadcast()

	switch {
	case !emptyServer && hasRequestedIP && clientIPZero:
		return SELECTING
	case emptyServer && hasRequestedIP && clientIPZero:
		return INIT_REBOOT
	case emptyServer && !clientIPZero:
		if isBroadcast {
			return REBINDING
		}
		return RENEWING
	default:
		slog.Warn("Unexpected packet state", "SIAddr", packet.SIAddr, "CIAddr", packet.CIAddr, "HasRequestedIP", hasRequestedIP)
		return InvalidState
	}
}
func MACToUint64(mac net.HardwareAddr) uint64 {
	return uint64(mac[0])<<40 | uint64(mac[1])<<32 | uint64(mac[2])<<24 | uint64(mac[3])<<16 | uint64(mac[4])<<8 | uint64(mac[5])
}

func IPToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

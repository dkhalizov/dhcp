package dhcp

import (
	"encoding/binary"
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"
)

type Server struct {
	bindings  map[uint64]*binding
	allocated map[uint64]*binding
	addrV4    net.IP
	gateway   net.IP
	macAddr   net.HardwareAddr
	config    *Config

	multicast *net.UDPConn
	conn      net.PacketConn
}

type Config struct {
	Start   net.IP
	End     net.IP
	Lease   time.Duration
	DNS1    net.IP
	DNS2    net.IP
	Name    []byte
	Subnet  net.IP
	Addr    net.IP
	Gateway net.IP
}

func (c *Config) validate() error {
	if c.Start == nil {
		return fmt.Errorf("start address is required")
	}
	if c.End == nil {
		return fmt.Errorf("end address is required")
	}
	if c.Lease == 0 {
		return fmt.Errorf("lease duration is required")
	}
	if c.DNS1 == nil {
		return fmt.Errorf("DNS1 is required")
	}
	if c.DNS2 == nil {
		return fmt.Errorf("DNS2 is required")
	}
	if c.Name == nil {
		return fmt.Errorf("name is required")
	}
	if c.Subnet == nil {
		return fmt.Errorf("subnet is required")
	}
	if c.Addr == nil {
		return fmt.Errorf("addr is required")
	}
	if c.Gateway == nil {
		return fmt.Errorf("gateway is required")
	}
	return nil
}

type binding struct {
	IP         net.IP
	MAC        net.HardwareAddr
	expiration time.Time
}

type Offer struct {
	ClientMAC net.HardwareAddr
	OfferIP   net.IP
	ServerIP  net.IP
}

func NewServer(cfg *Config) *Server {
	//validate config
	if err := cfg.validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	s := &Server{
		bindings:  make(map[uint64]*binding),
		allocated: make(map[uint64]*binding),
		config:    cfg,
		addrV4:    cfg.Addr,
		gateway:   cfg.Gateway,
	}

	conn, err := s.buildConn()
	if err != nil {
		log.Fatalf("Failed to build conn: %v", err)
		return nil
	}
	s.conn = conn
	return s
}

func (s *Server) Run() {
	slog.Info("Starting DHCP server")

	dial, err := net.Dial("udp", "255.255.255.255:68")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	s.multicast = dial.(*net.UDPConn)
	defer s.multicast.Close()
	slog.Info("DHCP server started on", "addr", s.multicast.LocalAddr())
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return
	}
	defer conn.Close()
	slog.Info("Listening on", "addr", conn.LocalAddr())
	for {
		data := make([]byte, 1024)
		n, peer, err := conn.ReadFromUDP(data)
		if err != nil {
			print(err)
			return
		}
		isUnicast := !peer.IP.Equal(net.IPv4bcast) && !peer.IP.IsMulticast()

		go s.processRawPacket(data[:n], isUnicast)
	}
}

func (s *Server) processRawPacket(data []byte, unicast bool) {
	p, err := Decode(data)
	if err != nil {
		slog.Error("failed to decode packet:", "err", err)
		return
	}
	dhcpMessageType := p.DHCPMessageType()
	slog.Info("received DHCP", "type", dhcpMessageType, "addr", p.CHAddr)
	var response *Packet
	switch dhcpMessageType {

	case DHCPDISCOVER:
		if err = s.discover(p); err != nil {
			slog.Error("Failed to send offer:", "err", err)
			return
		}
	case DHCPREQUEST:
		response = s.handleRequest(p, unicast)
	}
	if response == nil {
		return
	}
	var sendAddr *net.UDPAddr
	if p.GIAddr.Equal(net.IPv4zero) {
		// Direct communication with client
		if response.DHCPMessageType() == DHCPNAK {
			// NAK should be broadcast
			sendAddr = &net.UDPAddr{IP: net.IPv4bcast, Port: 68}
		} else if unicast && !response.CIAddr.Equal(net.IPv4zero) {
			// Unicast to the newly assigned address
			sendAddr = &net.UDPAddr{IP: response.CIAddr, Port: 68}
		} else {
			// Broadcast
			sendAddr = &net.UDPAddr{IP: net.IPv4bcast, Port: 68}
		}
	} else {
		// Communication via relay agent
		sendAddr = &net.UDPAddr{IP: p.GIAddr, Port: 68}
	}
	payload := response.Encode()
	raw := &Ethernet{
		SourcePort:      67,
		DestinationPort: 68,
		SourceIP:        s.addrV4,
		DestinationIP:   sendAddr.IP,
		DestinationMAC:  p.CHAddr,
		Payload:         payload,
	}
	err = s.Write(raw, sendAddr)
	if err != nil {
		slog.Error("Failed to send response:", "err", err)
	}
}

func (s *Server) nakOpts() *replyOpt {
	return &replyOpt{
		dhcpSrv: s.addrV4,
	}
}

func (s *Server) handleRequest(packet *Packet, unicast bool) *Packet {
	serverIdentifier := packet.ParsedOptions[OptionServerIdentifier]
	requestedIP := packet.ParsedOptions[OptionRequestedIPAddress]
	ciaddr := packet.CIAddr

	// SELECTING state
	if serverIdentifier != nil {
		slog.Info("Requesting IP", "addr", ciaddr.String(), "mac", packet.CHAddr)
		if !net.IP(serverIdentifier).Equal(s.config.Addr) {
			// Request not for this server
			return nil
		}
		if requestedIP == nil {
			packet.toNak(s.nakOpts())
			return packet
		}
		return s.handleSelectingRequest(packet, requestedIP)
	}

	// INIT-REBOOT state
	if requestedIP != nil && ciaddr.Equal(net.IPv4zero) {
		return s.handleInitRebootRequest(packet, requestedIP)
	}

	// RENEWING state
	if requestedIP == nil && !ciaddr.Equal(net.IPv4zero) && unicast {
		return s.handleRenewingRequest(packet, ciaddr)
	}

	// REBINDING state
	if requestedIP == nil && !ciaddr.Equal(net.IPv4zero) && !unicast {
		return s.handleRebindingRequest(packet, ciaddr)
	}

	return nil
}

func (s *Server) handleSelectingRequest(packet *Packet, requestedIP net.IP) *Packet {
	//if !s.isIPAvailable(requestedIP) {
	//	packet.toNak(s.nakOpts())
	//	return packet
	//}
	//
	//lease := Lease{
	//	IP:        requestedIP,
	//	MAC:       packet.ClientMAC,
	//	ExpiresAt: time.Now().Add(s.leaseTime),
	//}
	//s.leases[packet.ClientMAC.String()] = lease

	opt := s.defaultReplyOpt()
	opt.AddLease(s.config.Lease)
	packet.toAck(requestedIP, opt)

	return packet
}

func (s *Server) handleInitRebootRequest(packet *Packet, requestedIP net.IP) *Packet {
	if !s.inSubnet(requestedIP) {
		packet.toNak(s.nakOpts())
		return packet
	}

	//lease, exists := s.leases[packet.ClientMAC.String()]
	//if !exists || !lease.IP.Equal(requestedIP) {
	//	packet.toNak(s.nakOpts())
	//	return packet
	//}
	packet.toAck(requestedIP, s.defaultReplyOpt())
	return packet
}

func (s *Server) handleRenewingRequest(packet *Packet, ciaddr net.IP) *Packet {
	//lease, exists := s.leases[packet.ClientMAC.String()]
	//if !exists || !lease.IP.Equal(ciaddr) {
	//	packet.toNak(s.nakOpts())
	//	return packet
	//}
	//
	//lease.ExpiresAt = time.Now().Add(s.leaseTime)
	//s.leases[packet.ClientMAC.String()] = lease

	packet.toAck(ciaddr, s.defaultReplyOpt())
	return packet
}

func (s *Server) handleRebindingRequest(packet *Packet, ciaddr net.IP) *Packet {
	// Similar to handleRenewingRequest, but we need to check if this server has authority
	// In a multi-server setup, you might need additional logic here
	return s.handleRenewingRequest(packet, ciaddr)
}

func (s *Server) defaultReplyOpt() *replyOpt {
	return &replyOpt{
		router:        s.gateway,
		dhcpSrv:       s.addrV4,
		sName:         s.config.Name,
		subnet:        s.config.Subnet,
		renewTime:     [4]byte{0, 0, 0x0E, 0x10},
		rebindingTime: [4]byte{0, 0, 0x18, 0x9c},
		dns:           [8]byte{s.config.DNS1[0], s.config.DNS1[1], s.config.DNS1[2], s.config.DNS1[3], s.config.DNS2[0], s.config.DNS2[1], s.config.DNS2[2], s.config.DNS2[3]},
	}
}

func (s *Server) discover(p *Packet) error {
	bind := s.validateOffer(p)
	if bind == nil {
		slog.Debug("No available address")
		//send NAK
		return nil
	}
	slog.Info("Offering IP", "app", bind.IP, "addr", p.CHAddr.String())

	p.ToOffer(bind.IP, s.defaultReplyOpt())
	slog.Info("offering with", "packet", p)

	err := s.sendOffer(p)
	if err != nil {
		slog.Error("Failed to send unicast:", "err,", err)
	}
	return err
}

func (s *Server) reserve(addr net.IP, mac net.HardwareAddr) {
	b := binding{
		IP:         addr,
		MAC:        mac,
		expiration: time.Now().Add(time.Hour),
	}
	s.allocated[convertIPToUint64(addr)] = &b
	s.bindings[convertMACToUint64(mac)] = &b
}

func (s *Server) validateOffer(packet *Packet) *binding {
	now := time.Now()
	if b, ok := s.bindings[convertMACToUint64(packet.CHAddr)]; ok {
		// The client's current address as recorded in the client's current
		//        binding, ELSE
		if b.expiration.After(now) {
			return b
		}
		b.expiration = now.Add(s.config.Lease)
		// The client's previous address as recorded in the client's (now
		// expired or released) binding, if that address is in the server's
		// pool of available addresses and not already allocated, ELSE
		if _, ok = s.allocated[convertIPToUint64(b.IP)]; !ok {
			return b
		}
	}

	//      o The address requested in the 'Requested IP Address' option, if that
	//        address is valid and not already allocated, ELSE
	if o, ok := packet.ParsedOptions[OptionRequestedIPAddress]; ok {
		requested := net.IP{o[0], o[1], o[2], o[3]}
		if _, ok = s.allocated[convertIPToUint64(requested)]; !ok {
			return &binding{IP: requested, MAC: packet.CHAddr, expiration: now.Add(s.config.Lease)}
		}
	}

	//      o A new address allocated from the server's pool of available
	//        addresses; the address is selected based on the subnet from which
	//        the message was received (if 'giaddr' is 0) or on the address of
	//        the relay agent that forwarded the message ('giaddr' when not 0).
	start := s.config.Start
	end := s.config.End

	for ip := start; ip[3] <= end[3]; ip[3]++ {
		if _, ok := s.allocated[convertIPToUint64(ip)]; !ok {
			return &binding{IP: ip, MAC: packet.CHAddr, expiration: now.Add(s.config.Lease)}
		}
	}
	// If no address could be allocated,
	return nil
}

func (s *Server) Broadcast(p []byte) error {
	_, err := s.multicast.Write(p)
	return err
}

func (s *Server) inSubnet(addr []byte) bool {
	parsedIP := net.IP(addr)
	mask := net.IPMask(s.config.Subnet.To4())
	ones, _ := mask.Size()
	_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", s.config.Addr, ones))
	if err != nil {
		slog.Error("Failed to parse subnet:", "err", err)
		return false
	}
	return ipNet.Contains(parsedIP)
}

func convertMACToUint64(mac net.HardwareAddr) uint64 {
	var macUint64 uint64
	macBytes := mac[:6] // MAC addresses are 6 bytes long
	macUint64 = binary.BigEndian.Uint64(append([]byte{0, 0}, macBytes...))
	return macUint64
}

func convertIPToUint64(ip net.IP) uint64 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint64(append([]byte{0, 0, 0, 0}, ip...))
}

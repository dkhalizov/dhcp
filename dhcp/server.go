package dhcp

import (
	"encoding/binary"
	"errors"
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
		n, _, err := conn.ReadFromUDP(data)
		if err != nil {
			print(err)
			return
		}

		go s.processRawPacket(data[:n])
	}
}

func (s *Server) processRawPacket(data []byte) {
	p, err := Decode(data)
	if err != nil {
		slog.Error("failed to decode packet:", "err", err)
		return
	}
	dhcpMessageType := p.DHCPMessageType()
	slog.Info("received DHCP", "type", dhcpMessageType, "addr", p.CHAddr)
	switch dhcpMessageType {
	case DHCPDISCOVER:
		if err = s.discover(p); err != nil {
			slog.Error("Failed to send offer:", "err", err)
			return
		}
	case DHCPREQUEST:
		if err = s.request(p); err != nil {
			if err != errNak {
				slog.Error("Failed to send ACK:", "err", err)
				return
			}

			if net.IPv4zero.Equal(p.GIAddr) {
				//      If 'giaddr' is 0x0 in the DHCPREQUEST message, the client is on
				//      the same subnet as the server.  The server MUST broadcast the
				//      DHCPNAK message to the 0xffffffff broadcast address because the
				//      client may not have a correct network address or subnet mask, and
				//      the client may not be answering ARP requests.
			} else {
				//      If 'giaddr' is set in the DHCPREQUEST message, the client is on a
				//      different subnet.  The server MUST set the broadcast bit in the
				//      DHCPNAK, so that the relay agent will broadcast the DHCPNAK to the
				//      client, because the client may not have a correct network address
				//      or subnet mask, and the client may not be answering ARP requests.
				p.SetBroadcast()
			}

			p.toNak(&replyOpt{
				dhcpSrv: s.addrV4,
			})
			ack := p.Encode()
			err = s.Broadcast(ack)
			if err != nil {
				slog.Error("Failed to send NAK:", "err", err)
			}
			return
		}

	case DHCPDECLINE:

	case DHCPRELEASE:

	case DHCPINFORM:

	}
}

var errNak = errors.New("nak")

func (s *Server) request(p *Packet) error {
	var addr []byte
	var ok bool
	if addr, ok = p.HasOption(OptionServerIdentifier); !ok {
		// INIT-REBOOT
		//'server identifier' MUST NOT be filled in, 'requested IP address'
		//      option MUST be filled in with client's notion of its previously
		//      assigned address. 'ciaddr' MUST be zero. The client is seeking to
		//      verify a previously allocated, cached configuration. Server SHOULD
		//      send a DHCPNAK message to the client if the 'requested IP address'
		//      is incorrect, or is on the wrong network.
		if addr, ok = p.HasOption(OptionRequestedIPAddress); !ok {
			if !net.IPv4zero.Equal(p.CIAddr) {
				//RENWING
				// 'server identifier' MUST NOT be filled in, 'requested IP address'
				//      option MUST NOT be filled in, 'ciaddr' MUST be filled in with
				//      client's IP address. In this situation, the client is completely
				//      configured, and is trying to extend its lease. This message will
				//      be unicast, so no relay agents will be involved in its
				//      transmission.  Because 'giaddr' is therefore not filled in, the
				//      DHCP server will trust the value in 'ciaddr', and use it when
				//      replying to the client.
				if !s.inSubnet(p.CIAddr) {
					return errNak
				}
				//todo
				//return ask
			}
			return errNak
		}
		if !s.inSubnet(addr) || !s.inSubnet(p.GIAddr) {
			// Determining whether a client in the INIT-REBOOT state is on the
			//      correct network is done by examining the contents of 'giaddr', the
			//      'requested IP address' option, and a database lookup. If the DHCP
			//      server detects that the client is on the wrong net (i.e., the
			//      result of applying the local subnet mask or remote subnet mask (if
			//      'giaddr' is not zero) to 'requested IP address' option value
			//      doesn't match reality), then the server SHOULD send a DHCPNAK
			//      message to the client.
			return errNak
		}

		// If the network is correct, then the DHCP server should check if
		//      the client's notion of its IP address is correct. If not, then the
		//      server SHOULD send a DHCPNAK message to the client. If the DHCP
		//      server has no record of this client, then it MUST remain silent,
		//      and MAY output a warning to the network administrator. This
		//      behavior is necessary for peaceful coexistence of non-
		//      communicating DHCP servers on the same wire.
		//todo check notion

	}
	// SELECTING
	slog.Info("Requesting IP", "addr", p.CIAddr.String(), "mac", p.CHAddr.String())
	if !s.addrV4.Equal(addr) {
		// we are not the right server, drop
		return errNak
	}
	if !net.IPv4zero.Equal(p.CIAddr) {
		//'ciaddr' MUST be zero
		return errNak
	}

	if addr, ok = p.HasOption(OptionRequestedIPAddress); !ok {
		//??
		return errNak
	}
	// todo check and sync
	s.reserve(addr, p.CHAddr)

	opt := s.defaultReplyOpt()
	opt.AddLease(s.config.Lease)

	p.toAck(addr, opt)
	ack := p.Encode()
	raw := &Ethernet{
		SourcePort:      67,
		DestinationPort: 68,
		SourceIP:        s.addrV4,
		DestinationIP:   addr,
		DestinationMAC:  p.CHAddr,
		Payload:         ack,
	}
	_, err := s.conn.WriteTo(raw.Bytes(), &net.UDPAddr{IP: addr, Port: 68})
	return err
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
	_, ipNet, err := net.ParseCIDR(s.config.Subnet.String())
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

func compareIPv4(a, b net.IP) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

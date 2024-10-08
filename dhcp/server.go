package dhcp

import (
	"encoding/binary"
	"fmt"
	"log"
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

	udpConn *net.UDPConn
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

func NewServer(cfg *Config) *Server {
	//validate config
	if err := cfg.validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	return &Server{
		bindings:  make(map[uint64]*binding),
		allocated: make(map[uint64]*binding),
		config:    cfg,
		addrV4:    cfg.Addr,
		gateway:   cfg.Gateway,
	}
}

func (s *Server) Run() {
	log.Println("Starting DHCP server")
	dial, err := net.Dial("udp", "255.255.255.255:68")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	s.udpConn = dial.(*net.UDPConn)
	defer s.udpConn.Close()
	log.Printf("DHCP server started on %s", s.udpConn.LocalAddr())
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return
	}
	defer conn.Close()
	log.Printf("Listening on %s", conn.LocalAddr())
	for {
		data := make([]byte, 1024)
		_, _, err = conn.ReadFromUDP(data)
		if err != nil {
			print(err)
			return
		}
		s.processRawPacket(data)
	}
}

func (s *Server) processRawPacket(data []byte) {
	p, _ := decode(data)
	//p.Print()
	log.Printf("Received DHCP %b from %s\n", p.DHCPMessageType(), p.CHAddr)
	switch p.DHCPMessageType() {
	case DHCPDISCOVER:
		bind := s.validateOffer(p)
		if bind == nil {
			print("No available address")
			//send NAK
			return
		}
		log.Printf("Offering IP %s to %s\n", bind.IP, p.CHAddr)
		opt := replyOpt{
			router:        s.gateway,
			dhcpSrv:       s.addrV4,
			sName:         s.config.Name,
			subnet:        s.config.Subnet,
			renewTime:     [4]byte{0, 0, 0x0E, 0x10},
			rebindingTime: [4]byte{0, 0, 0x18, 0x9c},
			lease:         [4]byte{0, 0, 0x0E, 0x10}, //10 minutes
			dns:           [8]byte{s.config.DNS1[0], s.config.DNS1[1], s.config.DNS1[2], s.config.DNS1[3], s.config.DNS2[0], s.config.DNS2[1], s.config.DNS2[2], s.config.DNS2[3]},
		}
		log.Printf("offering with %v\n", opt)
		p.ToOffer(bind.IP, &opt)
		//send OFFER
		err := s.sendOffer(p, bind.MAC)
		if err != nil {
			fmt.Printf("Failed to send unicast: %v\n", err)
		}
	case DHCPREQUEST:
		var addr []byte
		var ok bool
		if addr, ok = p.HasOption(OptionServerIdentifier); !ok {
			// INIT-REBOOT
			// RENEWING
			// REBINDING
			return
		}
		// SELECTING
		if !compareIPv4(addr, s.addrV4) {
			// we are not the right server, drop
			return
		}
		// todo change to <<
		if convertIPToUint64(p.CIAddr) != 0 {
			//'ciaddr' MUST be zero
			return
		}
		//
		if addr, ok = p.HasOption(OptionRequestedIPAddress); !ok {
			//??
			return
		}
		// todo check and sync
		s.reserve(addr, p.CHAddr)

		//todo to config
		opt := replyOpt{
			router:        s.addrV4,
			dhcpSrv:       s.addrV4,
			sName:         s.config.Name,
			subnet:        s.config.Subnet,
			renewTime:     [4]byte{0, 0, 0x0E, 0x10},
			rebindingTime: [4]byte{0, 0, 0x18, 0x9c},
			dns:           [8]byte{s.config.DNS1[0], s.config.DNS1[1], s.config.DNS1[2], s.config.DNS1[3], s.config.DNS2[0], s.config.DNS2[1], s.config.DNS2[2], s.config.DNS2[3]},
		}
		p.toAck(addr, &opt)
		ack := p.Encode()
		raw := &Ethernet{
			SourcePort:      []byte{0, 67},
			DestinationPort: []byte{0, 68},
			SourceIP:        s.addrV4,
			DestinationIP:   addr,
			DestinationMAC:  p.CHAddr,
			Payload:         ack,
		}

		err := s.Unicast(raw)
		if err != nil {
			fmt.Printf("Failed to send unicast: %v\n", err)
		}

	case DHCPDECLINE:

	case DHCPRELEASE:

	case DHCPINFORM:

	}
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
	_, err := s.udpConn.Write(p)
	return err
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

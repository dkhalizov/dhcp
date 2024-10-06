package pkg

import (
	"fmt"
	"net"
	"time"
)

func (s *Server) processRawPacket(data []byte) {
	packet, _ := decode(data)
	packet.Print()
	switch packet.DHCPMessageType() {
	case DHCPDISCOVER:
		bind := s.discoverHandler(packet)
		if bind == nil {
			print("No available address")
			//send NAK
			return
		}
		packet.ToOffer(bind.IP)
		//send OFFER
		err := s.sendOffer(packet, bind.MAC)
		if err != nil {
			fmt.Printf("Failed to send unicast: %v\n", err)
		}
	case DHCPREQUEST:

	case DHCPDECLINE:

	case DHCPRELEASE:

	case DHCPINFORM:

	}
}

func (s *Server) discoverHandler(packet *Packet) *binding {
	now := time.Now()
	if b, ok := s.bindings[packet.CHAddr.String()]; ok {
		// The client's current address as recorded in the client's current
		//        binding, ELSE
		if b.expiration.After(now) {
			return b
		}
		b.expiration = now.Add(s.config.Lease)
		// The client's previous address as recorded in the client's (now
		// expired or released) binding, if that address is in the server's
		// pool of available addresses and not already allocated, ELSE
		if _, ok = s.allocated[b.IP.String()]; !ok {
			return b
		}
	}

	//      o The address requested in the 'Requested IP Address' option, if that
	//        address is valid and not already allocated, ELSE
	if o, ok := packet.ParsedOptions[50]; ok {
		requested := net.IP{o[0], o[1], o[2], o[3]}
		if _, ok = s.allocated[requested.String()]; !ok {
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
		if _, ok := s.allocated[ip.String()]; !ok {
			return &binding{IP: ip, MAC: packet.CHAddr, expiration: now.Add(s.config.Lease)}
		}
	}
	// If no address could be allocated,
	return nil
}

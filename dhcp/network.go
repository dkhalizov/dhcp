package dhcp

import (
	"log"
	"net"
)

func (s *Server) sendOffer(p *Packet, mac net.HardwareAddr) error {
	// broadcast flag
	if p.IsBroadcast() {
		return s.Broadcast(p.Encode())
	}
	log.Printf("Sending DHCP Offer to %s\n", p.YIAddr)
	offer := p.Encode()
	raw := &Ethernet{
		SourcePort:      []byte{0, 67},
		DestinationPort: []byte{0, 68},
		SourceIP:        p.SIAddr,
		DestinationIP:   p.YIAddr,
		SourceMAC:       s.macAddr,
		DestinationMAC:  mac,
		Payload:         offer,
	}
	return s.Unicast(raw)
}

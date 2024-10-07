package dhcp

import (
	"dhcp/dhcp/packet"
	"net"
)

func (s *Server) sendOffer(p *Packet, mac net.HardwareAddr) error {
	// if broadcast flag is set, send broadcast
	//todo implement broadcast
	//else send unicast
	//
	//offer := craftDHCPOffer(p.YIAddr, p.SIAddr, mac)
	offer := p.Encode()
	raw := &packet.Ethernet{
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

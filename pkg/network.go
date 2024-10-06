package pkg

import (
	"net"
)

func (s *Server) sendOffer(packet *Packet, mac net.HardwareAddr) error {
	// if broadcast flag is set, send broadcast
	//todo implement broadcast
	//else send unicast
	name, err := getInterfaceName()
	if err != nil {
		return err
	}
	//mac, packet.YIAddr, packet.SIAddr, name
	return s.Offer(&Offer{
		ClientMAC: mac,
		OfferIP:   packet.YIAddr,
		ServerIP:  packet.SIAddr,
		Interface: name,
	})
}

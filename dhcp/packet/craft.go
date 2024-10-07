package packet

import (
	"encoding/binary"
	"net"
)

type Ethernet struct {
	SourcePort, DestinationPort []byte
	SourceIP, DestinationIP     net.IP
	SourceMAC, DestinationMAC   net.HardwareAddr

	Payload []byte
}

func Craft(p *Ethernet) []byte {
	u := udp{
		Source:      binary.BigEndian.Uint16(p.SourcePort),
		Destination: binary.BigEndian.Uint16(p.DestinationPort),
		Length:      uint16(8 + len(p.Payload)),
		Payload:     p.Payload,
	}

	UDP := u.Encode()

	h := ipHeader{
		Version:     0x45, // IPv4
		Length:      uint16(20 + len(UDP)),
		Protocol:    17, // udp
		Source:      p.SourceIP,
		Destination: p.DestinationIP,
	}
	header := h.Encode()
	payload := append(header, UDP...)

	e := ethernet{
		Source:      p.SourceMAC,
		Destination: p.DestinationMAC,
		Type:        0x0800, // IPv4
		Payload:     payload,
	}
	encode := e.Encode()
	return encode
}

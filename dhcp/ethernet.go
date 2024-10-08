package dhcp

import (
	"encoding/binary"
	"net"
)

type ethernet struct {
	Source, Destination net.HardwareAddr
	Length              uint16
	Type                uint16

	Payload []byte
}

func (e *ethernet) Encode() []byte {
	eth := make([]byte, 14+len(e.Payload))
	copy(eth[0:6], e.Destination)
	copy(eth[6:12], e.Source)
	eth[12] = byte(e.Type >> 8)
	eth[13] = byte(e.Type)
	for i, b := range e.Payload {
		eth[i+14] = b
	}
	return eth
}

type Ethernet struct {
	SourcePort, DestinationPort []byte
	SourceIP, DestinationIP     net.IP
	SourceMAC, DestinationMAC   net.HardwareAddr

	Payload []byte
}

func (p *Ethernet) Bytes() []byte {
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

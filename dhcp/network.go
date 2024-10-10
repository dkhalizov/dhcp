package dhcp

import (
	"encoding/binary"
	"log/slog"
	"net"
)

type udp struct {
	Source, Destination uint16
	Length              uint16
	Payload             []byte
}

func (u *udp) Encode() []byte {
	data := make([]byte, 8+len(u.Payload))

	// Source port (67 for DHCP server)
	binary.BigEndian.PutUint16(data[0:], u.Source)

	// Destination port (68 for DHCP client)
	binary.BigEndian.PutUint16(data[2:], u.Destination)

	binary.BigEndian.PutUint16(data[4:], u.Length)
	for i, b := range u.Payload {
		data[i+8] = b
	}
	return data
}

type ipHeader struct {
	Version             uint8
	Length              uint16
	Protocol            uint8
	Source, Destination net.IP
}

func (i *ipHeader) Encode() []byte {
	data := make([]byte, 20)
	data[0] = i.Version
	binary.BigEndian.PutUint16(data[2:], i.Length)
	data[8] = 0xFF // TTL
	data[9] = i.Protocol
	copy(data[12:16], i.Source.To4())
	copy(data[16:20], i.Destination.To4())
	return data
}

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
	SourcePort, DestinationPort uint16
	SourceIP, DestinationIP     net.IP
	SourceMAC, DestinationMAC   net.HardwareAddr

	Payload []byte
}

func (p *Ethernet) Bytes() []byte {
	u := udp{
		Source:      p.SourcePort,
		Destination: p.DestinationPort,
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

func (p *Ethernet) udp() []byte {
	u := udp{
		Source:      p.SourcePort,
		Destination: p.DestinationPort,
		Length:      uint16(8 + len(p.Payload)),
		Payload:     p.Payload,
	}
	return u.Encode()
}

func (s *Server) sendOffer(p *Packet) error {
	//	broadcast flag
	if p.IsBroadcast() {
		slog.Info("Sending DHCP Offer to broadcast")
		return s.Broadcast(p.Encode())
	}
	slog.Info("Sending DHCP Offer to %s\n", p.YIAddr)
	offer := p.Encode()
	raw := &Ethernet{
		SourcePort:      67,
		DestinationPort: 68,
		SourceIP:        p.SIAddr,
		DestinationIP:   p.YIAddr,
		SourceMAC:       s.macAddr,
		DestinationMAC:  p.CHAddr,
		Payload:         offer,
	}

	return s.Write(raw, &net.UDPAddr{IP: p.YIAddr, Port: 68})
}

package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
)

const (
	ttlHeader        = 0xFF
	udpProtocol      = 17
	ethernetIPv4Type = 0x0800
	clientPort       = 68
	serverPort       = 67
)

type udp struct {
	Source, Destination uint16
	Length              uint16
	Payload             []byte
}

func (u *udp) Encode() []byte {
	data := make([]byte, 8+len(u.Payload))
	binary.BigEndian.PutUint16(data[0:], u.Source)
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
	data[8] = ttlHeader
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
		Protocol:    udpProtocol, // udp
		Source:      p.SourceIP,
		Destination: p.DestinationIP,
	}
	header := h.Encode()
	payload := append(header, UDP...)

	e := ethernet{
		Source:      p.SourceMAC,
		Destination: p.DestinationMAC,
		Type:        ethernetIPv4Type,
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

func SendPacket(conn net.PacketConn, p *Packet, sendAddr *net.UDPAddr) error {
	if conn == nil || p == nil {
		return errors.New("conn and packet must not be nil")
	}

	destAddr, err := resolveDestinationAddress(p, sendAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve destination address: %w", err)
	}

	encodedPacket := p.Encode()

	_, err = conn.WriteTo(encodedPacket, destAddr)
	if err != nil {
		slog.Error("Failed to send DHCP packet",
			"error", err,
			"client", p.CHAddr,
			"offer_ip", p.CIAddr,
			"destination", destAddr,
		)
		return fmt.Errorf("failed to send packet: %w", err)
	}

	slog.Debug("Sent DHCP packet",
		"client", p.CHAddr.String(),
		"offer_ip", p.CIAddr.String(),
		"destination", destAddr,
	)
	return nil
}
func resolveDestinationAddress(p *Packet, sendAddr *net.UDPAddr) (net.Addr, error) {
	if p.IsBroadcast() {
		return &net.UDPAddr{IP: net.IPv4bcast, Port: clientPort}, nil
	}

	if len(p.GetOption(OptionDHCPMessageType)) > 0 && p.GetOption(OptionDHCPMessageType)[0] == DHCPNAK {
		return &net.UDPAddr{IP: net.IPv4bcast, Port: clientPort}, nil
	}

	// If GIAddr is specified and not zero, send to the relay agent
	if p.GIAddr != nil && !p.GIAddr.IsUnspecified() {
		return &net.UDPAddr{IP: p.GIAddr, Port: serverPort}, nil
	}

	// Send directly to the client's IP if specified
	if p.CIAddr != nil && !p.CIAddr.IsUnspecified() {
		return &net.UDPAddr{IP: p.CIAddr, Port: clientPort}, nil
	}

	// Handle unicast
	if p.CHAddr != nil {
		//TODO
	}

	// Default to broadcast
	if sendAddr == nil || sendAddr.IP.IsUnspecified() {
		return &net.UDPAddr{IP: net.IPv4bcast, Port: clientPort}, nil
	}

	return sendAddr, nil
}

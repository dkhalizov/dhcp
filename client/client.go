package client

import (
	"bytes"
	"dhcp/protocol"
	"fmt"
	"net"
	"time"
)

const (
	serverPort = 67
	clientPort = 68
)

func createDHCPPacket(mac net.HardwareAddr, messageType byte, requestIP net.IP, serverIP net.IP) *protocol.Packet {
	packet := &protocol.Packet{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		XId:    uint32(time.Now().UnixNano()),
		Secs:   0,
		Flags:  0,
		CHAddr: mac,
	}

	var options []byte
	options = append(options, 53, 1, messageType)
	options = append(options, 61, byte(len(mac)+1), 1)
	options = append(options, mac...)
	if requestIP != nil {
		options = append(options, 50, 4)
		options = append(options, requestIP.To4()...)
	}
	if serverIP != nil {
		options = append(options, 54, 4)
		options = append(options, serverIP.To4()...)
	}
	options = append(options, 55, 4, 1, 3, 15, 6)
	options = append(options, 255)

	packet.Options = options
	return packet
}

func sendAndReceiveMultiple(packet *protocol.Packet, count int) ([]*protocol.Packet, error) {
	conn, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", clientPort))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	destAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", serverPort))
	if err != nil {
		return nil, err
	}

	_, err = conn.WriteTo(packet.Encode(), destAddr)
	if err != nil {
		return nil, err
	}

	responses := make([]*protocol.Packet, 0, count)
	for i := 0; i < count; i++ {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		buf := make([]byte, 1024)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				break
			}
			return nil, err
		}

		response, err := protocol.Decode(buf[:n])
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func chooseOffer(offers []*protocol.Packet) *protocol.Packet {
	if len(offers) == 0 {
		return nil
	}

	chosenOffer := offers[0]
	for _, offer := range offers[1:] {
		if bytes.Compare(offer.YIAddr[:], chosenOffer.YIAddr[:]) < 0 {
			chosenOffer = offer
		}
	}
	return chosenOffer
}

func getServerIP(packet *protocol.Packet) net.IP {
	for i := 0; i < len(packet.Options); i++ {
		if packet.Options[i] == 54 && i+5 < len(packet.Options) {
			return net.IP(packet.Options[i+2 : i+6])
		}
	}
	return nil
}

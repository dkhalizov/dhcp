package client

import (
	"bytes"
	"dhcp/dhcp"
	"fmt"
	"net"
	"time"
)

const (
	serverPort = 67
	clientPort = 68
)

func createDHCPPacket(mac net.HardwareAddr, messageType byte, requestIP net.IP, serverIP net.IP) *dhcp.Packet {
	packet := &dhcp.Packet{
		Op:     1, // BOOTREQUEST
		HType:  1, // Ethernet
		HLen:   6, // MAC address length
		Hops:   0,
		XID:    uint32(time.Now().UnixNano()),
		Secs:   0,
		Flags:  0,
		CHAddr: mac,
	}

	var options []byte
	// DHCP Message Type (Discover)
	options = append(options, 53, 1, messageType)
	// Client Identifier
	options = append(options, 61, byte(len(mac)+1), 1)
	options = append(options, mac...)
	if requestIP != nil {
		// Request IP Address
		options = append(options, 50, 4)
		options = append(options, requestIP.To4()...)
	}
	if serverIP != nil {
		// Server Identifier
		options = append(options, 54, 4)
		options = append(options, serverIP.To4()...)
	}
	// Parameter Request List
	options = append(options, 55, 4, 1, 3, 15, 6)
	options = append(options, 255) // End

	packet.Options = options
	return packet
}

func sendAndReceiveMultiple(packet *dhcp.Packet, count int) ([]*dhcp.Packet, error) {
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

	responses := make([]*dhcp.Packet, 0, count)
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

		response, err := dhcp.Decode(buf[:n])
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func chooseOffer(offers []*dhcp.Packet) *dhcp.Packet {
	if len(offers) == 0 {
		return nil
	}

	// For this example, we'll choose the offer with the lowest IP address
	// You can implement your own selection criteria here
	chosenOffer := offers[0]
	for _, offer := range offers[1:] {
		if bytes.Compare(offer.YIAddr[:], chosenOffer.YIAddr[:]) < 0 {
			chosenOffer = offer
		}
	}
	return chosenOffer
}

func getServerIP(packet *dhcp.Packet) net.IP {
	for i := 0; i < len(packet.Options); i++ {
		if packet.Options[i] == 54 && i+5 < len(packet.Options) { // Server Identifier option
			return net.IP(packet.Options[i+2 : i+6])
		}
	}
	return nil
}

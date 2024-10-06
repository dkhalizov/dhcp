package pkg

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type Packet struct {
	Op            byte
	HType         byte
	HLen          byte
	Hops          byte
	XID           uint32
	Secs          uint16
	Flags         uint16
	CIAddr        net.IP
	YIAddr        net.IP
	SIAddr        net.IP
	GIAddr        net.IP
	CHAddr        net.HardwareAddr
	SName         [64]byte
	File          [128]byte
	Options       []byte
	ParsedOptions map[byte][]byte
}

func (p *Packet) ToOffer(offer net.IP) {
	p.Op = 2
	p.YIAddr = offer
	p.SIAddr = []byte{192, 168, 2, 1}
	p.SName = [64]byte{ /* DHCP TEST */ 68, 72, 67, 80, 32, 84, 69, 83, 84}
	p.Options = []byte{
		53, 1, 2, // DHCP Message Type: Offer
		54, 4, 192, 168, 2, 1, // DHCP Server Identifier
		51, 4, 0, 0, 0, 60, // IP Address Lease Time
		1, 4, 255, 255, 255, 0, // Subnet Mask
		3, 4, 192, 168, 2, 1, // Router
		6, 4, 192, 168, 2, 1, // DNS
		255, // End
	}
}

func (p *Packet) DHCPMessageType() byte {
	return p.ParsedOptions[53][0]
}

func (p *Packet) Print() {
	fmt.Printf("Op: %d\n", p.Op)
	fmt.Printf("Hardware Type: %d\n", p.HType)
	fmt.Printf("Hardware Address Length: %d\n", p.HLen)
	fmt.Printf("Hops: %d\n", p.Hops)
	fmt.Printf("Transaction ID: %d\n", p.XID)
	fmt.Printf("Seconds: %d\n", p.Secs)
	fmt.Printf("Flags: %d\n", p.Flags)
	fmt.Printf("Client IP Address: %s\n", p.CIAddr)
	fmt.Printf("Your IP Address: %s\n", p.YIAddr)
	fmt.Printf("Server IP Address: %s\n", p.SIAddr)
	fmt.Printf("Gateway IP Address: %s\n", p.GIAddr)
	fmt.Printf("Client Hardware Address: %s\n", p.CHAddr)
	fmt.Printf("Server Name: %s\n", string(p.SName[:]))
	fmt.Printf("Boot Filename: %s\n", string(p.File[:]))
	fmt.Printf("Options: %v\n", getDHCPMessageType(p.Options))
}

func (p *Packet) Encode() []byte {
	data := make([]byte, 1024)
	data[0] = p.Op
	data[1] = p.HType
	data[2] = p.HLen
	data[3] = p.Hops
	binary.BigEndian.PutUint32(data[4:8], p.XID)
	binary.BigEndian.PutUint16(data[8:10], p.Secs)
	binary.BigEndian.PutUint16(data[10:12], p.Flags)
	copy(data[12:16], p.CIAddr.To4())
	copy(data[16:20], p.YIAddr.To4())
	copy(data[20:24], p.SIAddr.To4())
	copy(data[24:28], p.GIAddr.To4())
	copy(data[28:44], p.CHAddr)
	copy(data[44:108], p.SName[:])
	copy(data[108:236], p.File[:])

	i := 240
	for optN, opt := range p.ParsedOptions {
		data[i] = optN
		data[i+1] = byte(len(opt))
		copy(data[i+2:], opt)
		i += 2 + len(opt)
	}
	data[i] = 255
	return data
}

func decode(data []byte) (*Packet, error) {
	if len(data) < 240 {
		return nil, fmt.Errorf("packet too short")
	}

	packet := &Packet{
		Op:     data[0],
		HType:  data[1],
		HLen:   data[2],
		Hops:   data[3],
		XID:    binary.BigEndian.Uint32(data[4:8]),
		Secs:   binary.BigEndian.Uint16(data[8:10]),
		Flags:  binary.BigEndian.Uint16(data[10:12]),
		CIAddr: net.IP(data[12:16]),
		YIAddr: net.IP(data[16:20]),
		SIAddr: net.IP(data[20:24]),
		GIAddr: net.IP(data[24:28]),
		// todo allocate 6 bytes by default
		CHAddr:        make(net.HardwareAddr, 16),
		Options:       data[240:],
		ParsedOptions: make(map[byte][]byte),
	}

	copy(packet.CHAddr, data[28:44])
	copy(packet.SName[:], data[44:108])
	copy(packet.File[:], data[108:236])

	for i := 0; i < len(packet.Options); {
		optN := packet.Options[i]
		opt := DHCPOptions[optN]
		if opt.Name == "End" {
			break
		}
		size := int(packet.Options[i+1])
		packet.ParsedOptions[optN] = packet.Options[i+2 : i+2+size]
		i += size + 2
	}
	return packet, nil
}

func getDHCPMessageType(options []byte) string {
	b := strings.Builder{}
	for i := 0; i < len(options); {
		optN := options[i]
		opt := DHCPOptions[optN]
		if opt.Name == "End" {
			break
		}
		b.WriteString(opt.Name)
		b.WriteString(": ")
		size := int(options[i+1])
		for j := 0; j < size; j++ {
			if optN == 55 {
				b.WriteString(fmt.Sprintf("%s ", DHCPOptions[options[i+2+j]].Name))
			} else {
				b.WriteString(fmt.Sprintf("%d ", options[i+2+j]))
			}
		}

		i += int(options[i+1]) + 2

	}
	return b.String()
}

// craftIPHeader creates a simplified IP header for IPv4
func craftIPHeader(bodyLen int) []byte {
	ipHeader := make([]byte, 20)

	// IP Version and Header Length (IPv4)
	ipHeader[0] = 0x45

	// Total Length (will be filled later)
	totalLength := 20 + 8 + bodyLen // IP header + UDP header + DHCP payload
	binary.BigEndian.PutUint16(ipHeader[2:], uint16(totalLength))

	// Protocol (UDP = 17)
	ipHeader[9] = 17

	// Source IP
	ipHeader[12] = 192
	ipHeader[13] = 168
	ipHeader[14] = 1
	ipHeader[15] = 1

	// Destination IP
	ipHeader[16] = 192
	ipHeader[17] = 168
	ipHeader[18] = 1
	ipHeader[19] = 100

	// IP checksum (can be omitted for simplicity as NICs usually handle this)

	return ipHeader
}

// craftUDPHeader creates a simplified UDP header for DHCP Offer
func craftUDPHeader(bodyLen int) []byte {
	udpHeader := make([]byte, 8)

	// Source port (67 for DHCP server)
	binary.BigEndian.PutUint16(udpHeader[0:], 67)

	// Destination port (68 for DHCP client)
	binary.BigEndian.PutUint16(udpHeader[2:], 68)

	// Length (UDP header + DHCP payload)
	udpLength := 8 + bodyLen
	binary.BigEndian.PutUint16(udpHeader[4:], uint16(udpLength))

	return udpHeader
}

// craftDHCPOffer builds a DHCP Offer packet as payload
func craftDHCPOffer(offer net.IP, server net.IP, client net.HardwareAddr) []byte {
	dhcpOffer := make([]byte, 240) // DHCP packet minimum size

	// Message Type: Boot Reply (1 byte)
	dhcpOffer[0] = 0x02 // Boot Reply

	// Hardware Type: Ethernet (1 byte)
	dhcpOffer[1] = 0x01

	// Hardware Address Length (1 byte)
	dhcpOffer[2] = 0x06

	// Hops (1 byte)
	dhcpOffer[3] = 0x00

	// Transaction ID (4 bytes)
	binary.BigEndian.PutUint32(dhcpOffer[4:8], 0x12345678) // Example transaction ID

	// Seconds elapsed (2 bytes)
	binary.BigEndian.PutUint16(dhcpOffer[8:10], 0x0000)

	// Flags (2 bytes)
	binary.BigEndian.PutUint16(dhcpOffer[10:12], 0x0000)

	// Client IP (4 bytes) - Leave as 0 since it's unassigned
	binary.BigEndian.PutUint32(dhcpOffer[12:16], 0x00000000)

	// Your (Client) IP (4 bytes) - The offered IP
	binary.BigEndian.PutUint32(dhcpOffer[16:20], binary.BigEndian.Uint32(offer))

	// Server IP (4 bytes) - The DHCP server's IP
	binary.BigEndian.PutUint32(dhcpOffer[20:24], binary.BigEndian.Uint32(server))

	// Gateway IP (4 bytes) - Set to 0 if not used
	binary.BigEndian.PutUint32(dhcpOffer[24:28], 0x00000000)

	// Client MAC address (16 bytes)
	copy(dhcpOffer[28:44], client)

	// Server host name (64 bytes) - Set to 0 if not used
	for i := 44; i < 108; i++ {
		dhcpOffer[i] = 0
	}

	// Boot file name (128 bytes) - Set to 0 if not used
	for i := 108; i < 236; i++ {
		dhcpOffer[i] = 0
	}

	// Magic cookie (4 bytes)
	copy(dhcpOffer[236:240], []byte{99, 130, 83, 99})

	// DHCP options
	dhcpOptions := []byte{
		53, 1, 2, // DHCP Message Type = Offer (53, length 1, type 2)
		1, 4, 255, 255, 255, 0, // Subnet Mask option (255.255.255.0)
		3, 4, 192, 168, 1, 1, // Router option (192.168.1.1)
		54, 4, 192, 168, 1, 1, // DHCP Server Identifier
		51, 4, 0, 1, 81, 128, // Lease time (24 hours = 86400 seconds)
		255, // End option
	}

	// Add options to DHCP payload
	dhcpOffer = append(dhcpOffer, dhcpOptions...)

	return dhcpOffer
}

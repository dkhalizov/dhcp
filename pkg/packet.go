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

func decode(data []byte) (*Packet, error) {
	if len(data) < 240 {
		return nil, fmt.Errorf("packet too short")
	}

	packet := &Packet{
		Op:     data[0],
		HType:  data[1],
		HLen:   data[2],
		Hops:   data[3],
		XID:    binary.NativeEndian.Uint32(data[4:8]),
		Secs:   binary.NativeEndian.Uint16(data[8:10]),
		Flags:  binary.NativeEndian.Uint16(data[10:12]),
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

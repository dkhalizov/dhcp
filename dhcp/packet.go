package dhcp

import (
	o "dhcp/dhcp/options"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
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
	SName         []byte
	File          []byte
	Options       []byte
	ParsedOptions map[byte][]byte
}

func (p *Packet) HasOption(option byte) ([]byte, bool) {
	if v, ok := p.ParsedOptions[option]; ok {
		return v, true
	}
	return nil, false
}

func (p *Packet) IsBroadcast() bool {
	return p.Flags&0x8000 != 0
}

func (p *Packet) SetBroadcast() {
	p.Flags |= 0x8000
}

func (p *Packet) ToOffer(offer net.IP, opt *replyOpt) {
	p.Op = 2
	p.YIAddr = offer
	p.SIAddr = opt.dhcpSrv
	p.SName = opt.sName

	p.Options = []byte{
		OptionDHCPMessageType, 1, DHCPOFFER,
		OptionSubnetMask, byte(len(opt.subnet)), opt.subnet[0], opt.subnet[1], opt.subnet[2], opt.subnet[3],
		OptionRouter, byte(len(opt.router)), opt.router[0], opt.router[1], opt.router[2], opt.router[3],
		OptionServerIdentifier, byte(len(opt.dhcpSrv)), opt.dhcpSrv[0], opt.dhcpSrv[1], opt.dhcpSrv[2], opt.dhcpSrv[3],
		OptionIPAddressLeaseTime, byte(len(opt.lease)), opt.lease[0], opt.lease[1], opt.lease[2], opt.lease[3],
		OptionDomainNameServer, byte(len(opt.dns)), opt.dns[0], opt.dns[1], opt.dns[2], opt.dns[3], opt.dns[4], opt.dns[5], opt.dns[6], opt.dns[7],
		OptionEnd,
	}
}

func (p *Packet) DHCPMessageType() byte {
	if v, ok := p.ParsedOptions[OptionDHCPMessageType]; ok {
		return v[0]
	}
	return 0
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
	data := make([]byte, 240+len(p.Options))
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
	copy(data[236:240], magicCookie)
	copy(data[240:], p.Options)
	return data
}

func Decode(data []byte) (*Packet, error) {
	if len(data) < 240 {
		return nil, fmt.Errorf("packet too short")
	}

	packet := &Packet{
		Op:            data[0],
		HType:         data[1],
		HLen:          data[2],
		Hops:          data[3],
		XID:           binary.BigEndian.Uint32(data[4:8]),
		Secs:          binary.BigEndian.Uint16(data[8:10]),
		Flags:         binary.BigEndian.Uint16(data[10:12]),
		CIAddr:        net.IP(data[12:16]),
		YIAddr:        net.IP(data[16:20]),
		SIAddr:        net.IP(data[20:24]),
		GIAddr:        net.IP(data[24:28]),
		CHAddr:        data[28:44],
		SName:         data[44:108],
		File:          data[108:236],
		Options:       data[240:],
		ParsedOptions: make(map[byte][]byte),
	}

	for i := 0; i < len(packet.Options); {
		optN := packet.Options[i]
		if optN == 255 {
			break
		}
		size := int(packet.Options[i+1])
		packet.ParsedOptions[optN] = packet.Options[i+2 : i+2+size]
		i += size + 2
	}
	return packet, nil
}

var magicCookie = []byte{99, 130, 83, 99}

func getDHCPMessageType(options []byte) string {
	b := strings.Builder{}
	for i := 0; i < len(options); {
		optN := options[i]
		opt := o.DHCPOptions[optN]
		if opt.Name == "End" {
			break
		}
		b.WriteString(opt.Name)
		b.WriteString(": ")
		size := int(options[i+1])
		for j := 0; j < size; j++ {
			if optN == 55 {
				b.WriteString(fmt.Sprintf("%s ;", o.DHCPOptions[options[i+2+j]].Name))
			} else {
				b.WriteString(fmt.Sprintf("%d ;", options[i+2+j]))
			}
		}

		i += int(options[i+1]) + 2

	}
	return b.String()
}

type replyOpt struct {
	router        []byte
	dhcpSrv       []byte
	sName         []byte
	subnet        []byte
	lease         [4]byte
	renewTime     [4]byte
	rebindingTime [4]byte
	dns           [8]byte
}

func (r *replyOpt) AddLease(lease time.Duration) {
	binary.BigEndian.PutUint32(r.lease[:], uint32(lease.Seconds()))
}

func (p *Packet) toAck(offer net.IP, opt *replyOpt) {
	p.Op = 2
	p.YIAddr = offer
	p.SIAddr = opt.dhcpSrv
	p.SName = opt.sName

	//lease MUST (DHCPREQUEST) MUST NOT (DHCPINFORM) todo
	// DHCP options
	p.Options = []byte{
		OptionDHCPMessageType, 1, DHCPACK,
		OptionSubnetMask, 4, opt.subnet[0], opt.subnet[1], opt.subnet[2], opt.subnet[3],
		OptionRouter, 4, opt.router[0], opt.router[1], opt.router[2], opt.router[3],
		OptionServerIdentifier, 4, opt.dhcpSrv[0], opt.dhcpSrv[1], opt.dhcpSrv[2], opt.dhcpSrv[3],
		OptionIPAddressLeaseTime, 4, opt.lease[0], opt.lease[1], opt.lease[2], opt.lease[3],
		OptionRenewalTime, 4, opt.renewTime[0], opt.renewTime[1], opt.renewTime[2], opt.renewTime[3],
		OptionRebindingTime, 4, opt.rebindingTime[0], opt.rebindingTime[1], opt.rebindingTime[2], opt.rebindingTime[3],
		OptionDomainNameServer, 8, opt.dns[0], opt.dns[1], opt.dns[2], opt.dns[3], opt.dns[4], opt.dns[5], opt.dns[6], opt.dns[7],
		OptionEnd,
	}
}

func (p *Packet) toNak(opt *replyOpt) {
	p.Op = 2
	p.Options = []byte{
		OptionDHCPMessageType, 1, DHCPNAK,
		OptionServerIdentifier, 4, opt.dhcpSrv[0], opt.dhcpSrv[1], opt.dhcpSrv[2], opt.dhcpSrv[3],
		OptionEnd,
	}
}

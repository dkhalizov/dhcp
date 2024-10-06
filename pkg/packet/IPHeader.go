package packet

import "net"

type ipHeader struct {
	Version             uint8
	Length              uint16
	Protocol            uint8
	Source, Destination net.IP
}

func (i *ipHeader) Encode() []byte {
	ip := make([]byte, 20)
	ip[0] = i.Version
	ip[2] = uint8(i.Length >> 8)
	ip[3] = uint8(i.Length)
	ip[8] = 0xFF // TTL
	ip[9] = i.Protocol
	copy(ip[12:16], i.Source.To4())
	copy(ip[16:20], i.Destination.To4())
	return ip
}

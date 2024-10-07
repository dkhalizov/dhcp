package packet

import (
	"encoding/binary"
	"net"
)

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

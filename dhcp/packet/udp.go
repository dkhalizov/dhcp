package packet

import "encoding/binary"

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

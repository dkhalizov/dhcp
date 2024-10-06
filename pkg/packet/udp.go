package packet

type udp struct {
	Source, Destination uint16
	Length              uint16
	Payload             []byte
}

func (u *udp) Encode() []byte {
	data := make([]byte, 8+len(u.Payload))
	data[0] = byte(u.Source >> 8)
	data[1] = byte(u.Source)
	data[2] = byte(u.Destination >> 8)
	data[3] = byte(u.Destination)
	data[4] = byte(u.Length >> 8)
	data[5] = byte(u.Length)
	for i, b := range u.Payload {
		data[i+8] = b
	}
	return data
}

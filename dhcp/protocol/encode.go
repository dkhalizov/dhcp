package protocol

import (
	"encoding/binary"
	"net"
)

func flattenIPs(ips []net.IP) []byte {
	result := make([]byte, 0, len(ips)*4)
	for _, ip := range ips {
		result = append(result, ip.To4()...)
	}
	return result
}

func intToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// DHCP Discover Packet Creation in Array of Bytes
func buildDHCPDiscover() []byte {
	// Buffer to hold the entire DHCPDISCOVER packet
	var packet bytes.Buffer

	// Message Type: 1 = BOOTREQUEST
	packet.WriteByte(1) // op

	// Hardware Type: 1 = Ethernet
	packet.WriteByte(1) // htype

	// Hardware Address Length: 6 (for MAC)
	packet.WriteByte(6) // hlen

	// Hops: 0 (not used in DHCPDISCOVER)
	packet.WriteByte(0) // hops

	// Transaction ID: Random 4 bytes (can be random)
	xid := make([]byte, 4)
	rand.Seed(time.Now().UnixNano())
	binary.BigEndian.PutUint32(xid, rand.Uint32())
	packet.Write(xid) // xid

	// Seconds elapsed: 0
	packet.Write([]byte{0, 0}) // secs

	// Flags: Broadcast flag set to 0x8000
	packet.Write([]byte{0x80, 0x00}) // flags

	// Client IP address: 0.0.0.0 (since this is a DHCPDISCOVER)
	packet.Write([]byte{0, 0, 0, 0}) // ciaddr

	// 'Your' IP address, Server IP address, Gateway IP address all 0
	packet.Write([]byte{0, 0, 0, 0}) // yiaddr
	packet.Write([]byte{0, 0, 0, 0}) // siaddr
	packet.Write([]byte{0, 0, 0, 0}) // giaddr

	// Client MAC Address (example MAC address: 12:34:56:78:9A:BC)
	mac := net.HardwareAddr{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc}
	packet.Write(mac)

	// Fill remaining 10 bytes of client hardware address with zeros
	packet.Write(make([]byte, 10))

	// Server host name (64 bytes, all zeros)
	packet.Write(make([]byte, 64))

	// Boot file name (128 bytes, all zeros)
	packet.Write(make([]byte, 128))

	// Magic Cookie: DHCP
	packet.Write([]byte{99, 130, 83, 99}) // Magic cookie (DHCP)

	// DHCP Options
	options := []byte{
		53, 1, 1, // DHCP Message Type (53), length 1, value 1 (DHCPDISCOVER)
		55, 4, 1, 3, 6, 15, // Parameter Request List: Request subnet mask, router, DNS, domain name
		255, // End option
	}

	packet.Write(options)
	x := hex.EncodeToString(packet.Bytes())
	fmt.Printf(x)
	return packet.Bytes()
}

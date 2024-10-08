//go:build windows

package dhcp

import (
	"fmt"
	"syscall"
)

const (
	ETH_ALEN = 6
)

type EthernetHeader struct {
	Destination [ETH_ALEN]byte
	Source      [ETH_ALEN]byte
	Proto       uint16
}

func (s *Server) Unicast(p *Ethernet) error {
	if err := WSAStart(); err != nil {
		return err
	}
	defer syscall.WSACleanup()

	// Create a raw socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, 17)
	if err != nil {
		fmt.Println("Error creating socket:", err)
		return err
	}
	defer syscall.CloseHandle(fd)

	bytes := p.Bytes()
	// Prepare the WSABuf
	var wsaBuf syscall.WSABuf
	wsaBuf.Len = uint32(len(bytes))
	wsaBuf.Buf = &bytes[0]

	// Prepare the sockaddr structure for Windows
	var addr syscall.SockaddrInet4
	addr.Addr = [4]byte(p.DestinationIP)
	addr.Port = 68

	// Send the packet
	var bytesSent uint32
	err = syscall.WSASendto(fd, &wsaBuf, 1, &bytesSent, 0, &addr, nil, nil)
	if err != nil {
		fmt.Println("Error sending packet:", err)
		return err
	}

	fmt.Printf("Raw packet sent successfully, %d bytes sent\n", bytesSent)
	return nil
}

func WSAStart() error {
	var wsaData syscall.WSAData
	return syscall.WSAStartup(uint32(0x0202), &wsaData)
}

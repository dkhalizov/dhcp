//go:build linux

package dhcp

import (
	"fmt"
	"syscall"
)

func (s *Server) Unicast(p *Ethernet) error {
	iface, err := getInterface()
	if err != nil {
		return fmt.Errorf("failed to get interface: %v", err)
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		return fmt.Errorf("failed to create socket: %v", err)
	}
	defer syscall.Close(fd)

	if err := syscall.BindToDevice(fd, iface.Name); err != nil {
		return fmt.Errorf("failed to bind to device: %v", err)
	}

	p.SourceMAC = iface.HardwareAddr
	data := p.Bytes()

	var addr syscall.SockaddrLinklayer
	addr.Protocol = htons(syscall.ETH_P_IP)
	addr.Ifindex = iface.Index
	addr.Halen = 6
	copy(addr.Addr[:], p.DestinationMAC)

	err = syscall.Sendto(fd, data, 0, &addr)
	if err != nil {
		return fmt.Errorf("failed to send packet: %v", err)
	}

	fmt.Println("DHCP Offer frame sent successfully")
	return nil
}

func htons(host uint16) uint16 {
	return (host&0xff)<<8 | (host >> 8)
}

//go:build linux

package dhcp

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func (s *Server) Write(e *Ethernet) error {
	_, err := s.conn.WriteTo(e.Bytes(), &net.UDPAddr{IP: e.DestinationIP, Port: int(e.DestinationPort)})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) buildConn() (net.PacketConn, error) {
	iface, err := getInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %v", err)
	}

	fd, err := syscall.Socket(0x11, 0x3, int(htons(0x3)))
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %v", err)
	}
	defer syscall.Close(fd)

	f := os.NewFile(uintptr(fd), "")
	defer f.Close()

	if err = syscall.SetsockoptInt(fd, 0x1, 0x6, 1); err != nil {
		return nil, fmt.Errorf("cannot set broadcasting on socket: %v", err)
	}
	if err = syscall.SetsockoptInt(fd, 0x1, 0x2, 1); err != nil {
		return nil, fmt.Errorf("cannot set reuseaddr on socket: %v", err)
	}

	if err = syscall.BindToDevice(fd, iface.Name); err != nil {
		return nil, fmt.Errorf("failed to bind to device: %v", err)
	}
	conn, err := net.FilePacketConn(f)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func htons(host uint16) uint16 {
	return (host&0xff)<<8 | (host >> 8)
}

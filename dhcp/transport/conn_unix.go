//go:build linux

package transport

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

type UnixTransport struct {
	conn net.PacketConn
}

func (t *UnixTransport) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return t.conn.WriteTo(p, addr)
}

func (t *UnixTransport) Close() error {
	return t.conn.Close()
}

func (t *UnixTransport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *UnixTransport) SetDeadline(dt time.Time) error {
	return t.conn.SetDeadline(dt)
}

func (t *UnixTransport) SetReadDeadline(dt time.Time) error {
	return t.conn.SetReadDeadline(dt)
}

func (t *UnixTransport) SetWriteDeadline(dt time.Time) error {
	return t.conn.SetWriteDeadline(dt)
}

func (t *UnixTransport) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return t.conn.ReadFrom(p)
}

func BuildConn() (*UnixTransport, error) {
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
	return &UnixTransport{
		conn: conn,
	}, nil
}

func htons(host uint16) uint16 {
	return (host&0xff)<<8 | (host >> 8)
}

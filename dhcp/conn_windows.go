//go:build windows

package dhcp

import (
	"net"
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

func (s *Server) Write(e *Ethernet) error {
	_, err := s.conn.WriteTo(e.udp(), &net.UDPAddr{IP: e.DestinationIP, Port: 68})
	return err
}

func (s *Server) buildConn() (net.PacketConn, error) {
	conn, err := NewWinPacketConn()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func WSAStart() error {
	var wsaData syscall.WSAData
	return syscall.WSAStartup(uint32(0x0202), &wsaData)
}

var (
	modws2_32       = syscall.NewLazyDLL("ws2_32.dll")
	procSocket      = modws2_32.NewProc("socket")
	procBind        = modws2_32.NewProc("bind")
	procSendto      = modws2_32.NewProc("sendto")
	procRecvfrom    = modws2_32.NewProc("recvfrom")
	procSetsockopt  = modws2_32.NewProc("setsockopt")
	procClosesocket = modws2_32.NewProc("closesocket")
)

type WinPacketConn struct {
	fd syscall.Handle
}

func NewWinPacketConn() (*WinPacketConn, error) {
	fd, _, err := procSocket.Call(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_UDP)
	if err != nil && err != syscall.Errno(0) {
		return nil, err
	}

	_, _, err = procSetsockopt.Call(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, uintptr(unsafe.Pointer(&[]byte{1}[0])), 1)
	if err != nil && err != syscall.Errno(0) {
		syscall.CloseHandle(syscall.Handle(fd))
		return nil, err
	}

	_, _, err = procSetsockopt.Call(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, uintptr(unsafe.Pointer(&[]byte{1}[0])), 1)
	if err != nil && err != syscall.Errno(0) {
		syscall.CloseHandle(syscall.Handle(fd))
		return nil, err
	}

	return &WinPacketConn{fd: syscall.Handle(fd)}, nil
}

func (c *WinPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, errors.New("addr is not UDPAddr")
	}
	sa := syscall.RawSockaddrInet4{
		Family: syscall.AF_INET,
		Port:   uint16(udpAddr.Port<<8) | uint16(udpAddr.Port>>8), // network byte order
	}
	copy(sa.Addr[:], udpAddr.IP.To4())

	r1, _, e1 := procSendto.Call(
		uintptr(c.fd),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		0,
		uintptr(unsafe.Pointer(&sa)),
		unsafe.Sizeof(sa),
	)
	if r1 != 0 {
		return 0, e1
	}
	return int(r1), nil
}

func (c *WinPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	var from syscall.RawSockaddrInet4
	fromlen := int32(unsafe.Sizeof(from))
	r1, _, e1 := procRecvfrom.Call(
		uintptr(c.fd),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		0,
		uintptr(unsafe.Pointer(&from)),
		uintptr(unsafe.Pointer(&fromlen)),
	)
	if r1 != 0 {
		return 0, nil, e1
	}

	ip := net.IPv4(from.Addr[0], from.Addr[1], from.Addr[2], from.Addr[3])
	port := int(from.Port>>8) | int(from.Port<<8)
	return int(r1), &net.UDPAddr{IP: ip, Port: port}, nil
}

func (c *WinPacketConn) Close() error {
	_, _, err := procClosesocket.Call(uintptr(c.fd))
	return err
}

func (c *WinPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *WinPacketConn) SetDeadline(t time.Time) error {
	return c.setDeadline(t, 0)
}

func (c *WinPacketConn) SetReadDeadline(t time.Time) error {
	return c.setDeadline(t, 0)
}

func (c *WinPacketConn) SetWriteDeadline(t time.Time) error {
	return c.setDeadline(t, 0)
}

func (c *WinPacketConn) setDeadline(t time.Time, opt int) error {
	var tv int64
	if !t.IsZero() {
		d := t.Sub(time.Now())
		if d < 0 {
			tv = 1 // minimum non-zero timeout
		} else {
			tv = int64(d / time.Millisecond)
		}
	}
	_, _, err := procSetsockopt.Call(
		uintptr(c.fd),
		syscall.SOL_SOCKET,
		uintptr(opt),
		uintptr(unsafe.Pointer(&tv)),
		unsafe.Sizeof(tv),
	)
	if err != nil && err != syscall.Errno(0) {
		return err
	}
	return nil
}

//go:build arm64 && darwin

package dhcp

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func (s *Server) Write(e *Ethernet, addr net.Addr) error {
	_, err := s.conn.WriteTo(e.Bytes(), addr)
	return err
}

func findAvailableBPF() (int, error) {
	var bpf int
	var err error
	for i := 0; i < 10; i++ { // Check the first 10 BPF devices (bpf0 to bpf9)
		device := fmt.Sprintf("/dev/bpf%d", i)
		bpf, err = syscall.Open(device, syscall.O_RDWR, 0)
		if err == nil {
			return bpf, nil // Return the first available device
		}
		if !os.IsPermission(err) && !os.IsNotExist(err) {
			slog.Info("Device %s is busy, trying next...", device)
		}
	}
	return -1, fmt.Errorf("no available BPF devices found")
}

func enableImmediateMode(bpf int) {
	// Enable immediate mode (BIOCIMMEDIATE)
	var immediate int = 1
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(bpf), syscall.BIOCIMMEDIATE, uintptr(unsafe.Pointer(&immediate)))
	if errno != 0 {
		log.Fatalf("Failed to enable immediate mode: %v", errno)
	}
}

func bindToDevice(bpf int, ifaceName string) error {
	// Convert interface name to a byte array for system call
	var ifreq [syscall.IFNAMSIZ]byte
	copy(ifreq[:], ifaceName)

	// Use ioctl to bind BPF to the network interface
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(bpf), syscall.BIOCSETIF, uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		return fmt.Errorf("failed to bind to device: %v", errno)
	}
	return nil
}

func (s *Server) buildConn() (net.PacketConn, error) {
	return newBPFPacketConn()

}

func newBPFPacketConn() (net.PacketConn, error) {
	bpf, err := findAvailableBPF()
	if err != nil {
		log.Fatalf("Failed to open BPF device: %v", err)
	}
	defer syscall.Close(bpf)

	iface, err := getInterface()
	if err != nil {
		log.Fatalf("Failed to get interface: %v", err)
	}

	err = bindToDevice(bpf, iface.Name)
	if err != nil {
		log.Fatalf("Failed to bind to device: %v", err)
	}

	enableImmediateMode(bpf)
	slog.Info("Bound to device", "name", iface.Name)
	return &bpfPacketConn{fd: bpf, name: iface.Name}, nil
}

type bpfPacketConn struct {
	fd   int
	name string
}

func (b *bpfPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, err := syscall.Read(b.fd, p)
	if err != nil {
		return 0, nil, err
	}
	return n, nil, nil // BPF doesn't provide source address info
}

func (b *bpfPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return syscall.Write(b.fd, p)
}

func (b *bpfPacketConn) Close() error {
	return syscall.Close(b.fd)
}

func (b *bpfPacketConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4zero}
}

func (b *bpfPacketConn) SetDeadline(t time.Time) error {
	return nil // Not implemented
}

func (b *bpfPacketConn) SetReadDeadline(t time.Time) error {
	return nil // Not implemented
}

func (b *bpfPacketConn) SetWriteDeadline(t time.Time) error {
	return nil // Not implemented
}

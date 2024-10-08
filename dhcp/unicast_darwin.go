//go:build arm64 && darwin

package dhcp

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

func (s *Server) Unicast(p *Ethernet) error {
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
	p.SourceMAC = iface.HardwareAddr
	data := p.Bytes()
	_, err = syscall.Write(bpf, data)
	if err != nil {
		log.Fatalf("Failed to send packet: %v", err)
	}

	log.Println("DHCP Offer frame sent successfully")
	return nil
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
			log.Printf("Device %s is busy, trying next...", device)
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

//go:build arm64 && darwin

package dhcp

import (
	"dhcp/dhcp/packet"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

func (s *Server) Unicast(p *packet.Ethernet) error {
	intName, err := getInterfaceName()
	if err != nil {
		return err

	}
	bpf, err := findAvailableBPF()
	if err != nil {
		log.Fatalf("Failed to open BPF device: %v", err)
	}
	defer syscall.Close(bpf)

	iface, err := getInterface(intName)
	if err != nil {
		log.Fatalf("Failed to get interface: %v", err)
	}

	err = bindToDevice(bpf, iface.Name)
	if err != nil {
		log.Fatalf("Failed to bind to device: %v", err)
	}

	enableImmediateMode(bpf)
	p.SourceMAC = iface.HardwareAddr
	data := packet.Craft(p)
	_, err = syscall.Write(bpf, data)
	if err != nil {
		log.Fatalf("Failed to send packet: %v", err)
	}

	fmt.Println("DHCP Offer frame sent successfully")
	return nil
}

func (s *Server) Ack(o *Offer) error {
	bpf, err := findAvailableBPF()
	if err != nil {
		log.Fatalf("Failed to open BPF device: %v", err)
	}
	defer syscall.Close(bpf)

	iface, err := getInterface(o.Interface)
	if err != nil {
		log.Fatalf("Failed to get interface: %v", err)
	}

	err = bindToDevice(bpf, iface.Name)
	if err != nil {
		log.Fatalf("Failed to bind to device: %v", err)
	}

	enableImmediateMode(bpf)
	offer := craftDHCPAck(o.OfferIP, o.ServerIP, o.ClientMAC)
	p := &packet.Ethernet{
		SourcePort:      []byte{0, 67},
		DestinationPort: []byte{0, 68},
		SourceIP:        o.ServerIP,
		DestinationIP:   o.OfferIP,
		SourceMAC:       iface.HardwareAddr,
		DestinationMAC:  o.ClientMAC,
		Payload:         offer,
	}
	data := packet.Craft(p)

	_, err = syscall.Write(bpf, data)
	if err != nil {
		log.Fatalf("Failed to send packet: %v", err)
	}

	fmt.Println("DHCP Offer frame sent successfully")
	return nil
}

// findAvailableBPF loops through BPF devices and finds the first available one.
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

// enableImmediateMode enables immediate mode on the BPF device.
func enableImmediateMode(bpf int) {
	// Enable immediate mode (BIOCIMMEDIATE)
	var immediate int = 1
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(bpf), syscall.BIOCIMMEDIATE, uintptr(unsafe.Pointer(&immediate)))
	if errno != 0 {
		log.Fatalf("Failed to enable immediate mode: %v", errno)
	}
}

// bindToDevice binds the BPF file descriptor to a network interface.
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

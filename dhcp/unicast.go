package dhcp

import (
	"net"
)

type Offer struct {
	ClientMAC net.HardwareAddr
	OfferIP   net.IP
	ServerIP  net.IP
	Interface string
}

type Network interface {
	Unicast(o *Ethernet) error
}

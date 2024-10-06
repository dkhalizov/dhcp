package pkg

import "net"

type Offer struct {
	ClientMAC net.HardwareAddr
	OfferIP   net.IP
	ServerIP  net.IP
	Interface string
}

type Unicast interface {
	Offer(o *Offer) error
}

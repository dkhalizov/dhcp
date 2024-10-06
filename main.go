package main

import (
	"dhcp/pkg"
	"net"
	"time"
)

func main() {
	pkg.NewServer(&pkg.Config{
		Start: net.IPv4(192, 168, 2, 10),
		End:   net.IPv4(192, 168, 2, 20),
		Lease: time.Minute * 10,
	}).Run()
}

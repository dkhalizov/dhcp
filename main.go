package main

import (
	"dhcp/dhcp"
	"net"
	"time"
)

func main() {
	dhcp.NewServer(&dhcp.Config{
		Start: net.IPv4(192, 168, 2, 10),
		End:   net.IPv4(192, 168, 2, 20),
		Lease: time.Minute * 10,
	}).Run()
}

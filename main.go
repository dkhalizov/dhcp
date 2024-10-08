package main

import (
	"dhcp/dhcp"
	"time"
)

func main() {
	server := dhcp.NewServer(&dhcp.Config{
		Start:   []byte{172, 20, 0, 10},
		End:     []byte{72, 20, 0, 20},
		Lease:   time.Minute * 10,
		DNS1:    []byte{8, 8, 8, 8},
		DNS2:    []byte{8, 8, 4, 4},
		Name:    []byte{ /* DHCP TEST */ 68, 72, 67, 80, 32, 84, 69, 83, 84},
		Subnet:  []byte{255, 255, 0, 0},
		Addr:    []byte{172, 20, 0, 2},
		Gateway: []byte{172, 20, 0, 1},
	})
	server.Run()
}

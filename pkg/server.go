package pkg

import (
	"net"
	"time"
)

func Run() {

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		data := make([]byte, 1024)
		_, _, err := conn.ReadFromUDP(data)
		if err != nil {
			return
		}
		packet, _ := decode(data)
		packet.Print()

		switch packet.DHCPMessageType() {
		case DHCPDISCOVER:
			discoverHandler(packet)
		case DHCPREQUEST:

		case DHCPDECLINE:

		case DHCPRELEASE:

		case DHCPINFORM:

		}
	}

}

type binding struct {
	IP         net.IP
	expiration time.Time
}

var bindings = make(map[string]binding)

func discoverHandler(packet *Packet) binding {
	// The client's current address as recorded in the client's current
	//        binding, ELSE
	now := time.Now()
	if b, ok := bindings[packet.CHAddr.String()]; ok {
		if b.expiration.After(now) {
			return b
		}
		//      o The client's previous address as recorded in the client's (now
		//        expired or released) binding, if that address is in the server's
		//        pool of available addresses and not already allocated, ELSE

	}

	//      o The client's previous address as recorded in the client's (now
	//        expired or released) binding, if that address is in the server's
	//        pool of available addresses and not already allocated, ELSE
	//
	//      o The address requested in the 'Requested IP Address' option, if that
	//        address is valid and not already allocated, ELSE
	//
	//      o A new address allocated from the server's pool of available
	//        addresses; the address is selected based on the subnet from which
	//        the message was received (if 'giaddr' is 0) or on the address of
	//        the relay agent that forwarded the message ('giaddr' when not 0).
	return nil
}

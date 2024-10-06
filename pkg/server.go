package pkg

import (
	"net"
	"time"
)

type Server struct {
	bindings  map[string]*binding
	allocated map[string]*binding

	config *Config
}

type Config struct {
	Start net.IP
	End   net.IP
	Lease time.Duration
}

type binding struct {
	IP         net.IP
	MAC        net.HardwareAddr
	expiration time.Time
}

func NewServer(cfg *Config) *Server {
	return &Server{
		bindings:  make(map[string]*binding),
		allocated: make(map[string]*binding),
		config:    cfg,
	}
}

func (s *Server) Run() {
	go func() {
		conn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: 68})
		for {
			buffer := make([]byte, 1024)
			conn.ReadFromUDP(buffer)
			packet, _ := decode(buffer)
			packet.Print()
		}
	}()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		data := make([]byte, 1024)
		_, _, err = conn.ReadFromUDP(data)
		if err != nil {
			print(err)
			return
		}
		s.processRawPacket(data)
	}
}

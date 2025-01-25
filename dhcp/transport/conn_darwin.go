//go:build arm64 && darwin

package transport

import (
	"dhcp/protocol"
	"log/slog"
	"net"
)

func Write(conn net.PacketConn, e *protocol.Ethernet, addr net.Addr) error {
	_, err := conn.WriteTo(e.Bytes(), addr)
	return err
}

func BuildConn() (net.PacketConn, error) {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return nil, err
	}
	slog.Info("Listening on", "addr", udpConn.LocalAddr())
	//slog.Info("Broadcasting on", "addr", udpConn.RemoteAddr())
	return udpConn, nil
}

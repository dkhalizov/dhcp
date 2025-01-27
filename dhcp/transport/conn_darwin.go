//go:build arm64 && darwin

package transport

import (
	"log/slog"
	"net"
	"time"
)

type DarwinTransport struct {
	conn net.PacketConn
}

func (t *DarwinTransport) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return t.conn.WriteTo(p, addr)
}

func (t *DarwinTransport) Close() error {
	return t.conn.Close()
}

func (t *DarwinTransport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *DarwinTransport) SetDeadline(dt time.Time) error {
	return t.conn.SetDeadline(dt)
}

func (t *DarwinTransport) SetReadDeadline(dt time.Time) error {
	return t.conn.SetReadDeadline(dt)
}

func (t *DarwinTransport) SetWriteDeadline(dt time.Time) error {
	return t.conn.SetWriteDeadline(dt)
}

func (t *DarwinTransport) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return t.conn.ReadFrom(p)
}

func BuildConn() (*DarwinTransport, error) {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 67})
	if err != nil {
		return nil, err
	}
	slog.Info("Listening on", "addr", udpConn.LocalAddr())
	//slog.Info("Broadcasting on", "addr", udpConn.RemoteAddr())
	return &DarwinTransport{
		conn: udpConn,
	}, nil
}

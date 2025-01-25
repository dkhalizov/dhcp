package pool

import (
	"fmt"
	"net"
	"sync"
)

type IPPool struct {
	start     uint32
	end       uint32
	available []uint32
	m         sync.Mutex
}

func NewIPPool(start, end net.IP) (*IPPool, error) {
	startInt := ip4ToUint32(start)
	endInt := ip4ToUint32(end)
	if startInt > endInt {
		return nil, fmt.Errorf("invalid IP range")
	}

	pool := &IPPool{
		start:     startInt,
		end:       endInt,
		available: make([]uint32, 0, endInt-startInt+1),
	}

	for ip := startInt; ip <= endInt; ip++ {
		pool.available = append(pool.available, ip)
	}

	return pool, nil
}

func (p *IPPool) Allocate() net.IP {
	if len(p.available) == 0 {
		return nil
	}
	p.m.Lock()
	ip := p.available[0]
	p.available = p.available[1:]
	p.m.Unlock()
	return uint32ToIP4(ip)
}

func (p *IPPool) Release(ip net.IP) {
	ipInt := ip4ToUint32(ip)
	if ipInt >= p.start && ipInt <= p.end {
		p.m.Lock()
		p.available = append(p.available, ipInt)
		p.m.Unlock()
	}
}

func ip4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP4(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

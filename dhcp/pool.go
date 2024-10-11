package dhcp

import (
	"fmt"
	"net"
)

type IPPool struct {
	start     uint32
	end       uint32
	available []uint32
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

func (p *IPPool) allocate() net.IP {
	if len(p.available) == 0 {
		return nil
	}
	ip := p.available[0]
	p.available = p.available[1:]
	return uint32ToIP4(ip)
}

func (p *IPPool) release(ip net.IP) {
	ipInt := ip4ToUint32(ip)
	if ipInt >= p.start && ipInt <= p.end {
		p.available = append(p.available, ipInt)
	}
}

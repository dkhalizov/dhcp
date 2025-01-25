package transport

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
)

func getInterfaceName() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces that are down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return iface.Name, nil
				}
			}
		}

	}

	return "", errors.New("no suitable network interface found")
}

func getInterface() (*net.Interface, error) {
	ifaceName, err := getInterfaceName()
	if err != nil {
		return nil, err
	}
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("could not get interface: %v", err)
	}
	slog.Info("Using interface:", "name", iface.Name)
	return iface, nil
}

func GetMTU() (int, error) {
	iface, err := getInterface()
	if err != nil {
		return 0, err
	}
	return iface.MTU, nil
}

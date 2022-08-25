package net

import (
	"net"
	"strings"
)

// GetLocalIp interval ip addr
func GetLocalIp() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err.Error())
	}
	for _, addr := range addresses {
		if netIp, ok := addr.(*net.IPNet); ok && !netIp.IP.IsLoopback() &&
			netIp.IP.To4() != nil {
			return strings.Split(addr.String(), "/")[0]
		}
	}
	return ""
}

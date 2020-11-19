package helpers

import "net"

// IsIP check if a value is an IP or not
func IsIP(value string) bool {
	return net.ParseIP(value) != nil
}

// IsIPv4 check if a value is an IP v4
func IsIPv4(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6 check if a value is an IP v4
func IsIPv6(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil {
		return false
	}
	return ip.To16() != nil
}

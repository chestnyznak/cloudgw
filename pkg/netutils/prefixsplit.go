package netutils

import (
	"net"
)

// Addr returns IP address from the prefix and empty string for wrong prefix format
func Addr(prefix string) string {
	ip, _, err := net.ParseCIDR(prefix)
	if err != nil {
		return ""
	}

	return ip.String()
}

// MaskLen returns mask length from prefix and null string for wrong prefix format
func MaskLen(prefix string) uint32 {
	_, network, err := net.ParseCIDR(prefix)
	if err != nil {
		return 0
	}

	ones, _ := network.Mask.Size()

	return uint32(ones)
}

package netutils

import (
	"net"
	"strconv"
	"strings"
)

// MPLSLabel returns MPLS local label for an VRF as 10000000 + string(ipPrefix)[last 4 digits]
func MPLSLabel(ipPrefix string) (uint32, error) {
	ipAddr, _, err := net.ParseCIDR(ipPrefix)
	if err != nil {
		return 0, err
	}

	ipAddrWithoutDot := strings.ReplaceAll(ipAddr.String(), ".", "")

	ipAddrNum, err := strconv.Atoi(ipAddrWithoutDot)
	if err != nil {
		return 0, err
	}

	mplsLabel := uint32(1000000) + uint32(ipAddrNum%10000)

	return mplsLabel, nil
}

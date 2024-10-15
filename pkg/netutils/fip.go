package netutils

import (
	"net"
)

func IsFIP(fipPrefix string, parsedVPPAggregatedFIPs []*net.IPNet) bool {
	parsedFIPPrefix, _, err := net.ParseCIDR(fipPrefix)
	if err != nil {
		return false
	}

	for _, parsedVPPAggregatedFIP := range parsedVPPAggregatedFIPs {
		if parsedVPPAggregatedFIP.Contains(parsedFIPPrefix) {
			return true
		}
	}

	return false
}

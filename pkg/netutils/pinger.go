package netutils

import (
	"net"
	"time"

	"github.com/tatsushid/go-fastping"
)

// IsAddressAlive checks if IP address is available and return true if IP address is available
// and false if not pingable or the IP address is not correct.
func IsAddressAlive(ipAddress string, intervalInSec uint) (bool, error) {
	var isAlive bool

	p := fastping.NewPinger()
	p.MaxRTT = time.Duration(intervalInSec * 1_000_000_000)

	// Use UDP as network endpoint to avoid root permissions
	if _, err := p.Network("udp"); err != nil {
		return false, err
	}

	resolvedAddress, err := net.ResolveIPAddr("ip4:icmp", ipAddress)
	if err != nil {
		return false, err
	}

	p.AddIPAddr(resolvedAddress)

	p.OnRecv = func(_ *net.IPAddr, _ time.Duration) {
		isAlive = true
	}

	if err = p.Run(); err != nil {
		return false, err
	}

	return isAlive, nil
}

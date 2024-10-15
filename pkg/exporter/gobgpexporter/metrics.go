package gobgpexporter

import (
	"sync/atomic"
)

// GoBGPGeneralMetric describes general GoBGP metrics (update at runtime by BGP peer events)
type GoBGPGeneralMetric struct {
	VRFCount        int32
	PeerCount       int32
	ActivePeerCount int32
}

func (m *GoBGPGeneralMetric) IncVRFCount() {
	atomic.AddInt32(&m.VRFCount, 1)
}

func (m *GoBGPGeneralMetric) IncActivePeerCount() {
	atomic.AddInt32(&m.ActivePeerCount, 1)
}

func (m *GoBGPGeneralMetric) DecActivePeerCount() {
	atomic.AddInt32(&m.ActivePeerCount, -1)
}

var GoBGPGeneralMetrics GoBGPGeneralMetric

// GoBGPPerPeerMetric describes GoBGP BGP update counts per BGP peer. Updated by UpdateGoBGPPerPeerMetrics every n * sec
type GoBGPPerPeerMetric struct {
	IPv4RouteRcvd  float64
	IPv4RouteAdvd  float64
	Vpnv4RouteRcvd float64
	Vpnv4RouteAdvd float64
}

func NewGoBGPPerPeerMetric() *GoBGPPerPeerMetric {
	return &GoBGPPerPeerMetric{
		IPv4RouteRcvd:  0,
		IPv4RouteAdvd:  0,
		Vpnv4RouteRcvd: 0,
		Vpnv4RouteAdvd: 0,
	}
}

func (m *GoBGPPerPeerMetric) SetIPv4UpdateRcvd(value float64) {
	m.IPv4RouteRcvd = value
}

func (m *GoBGPPerPeerMetric) SetIPv4UpdateSent(value float64) {
	m.IPv4RouteAdvd = value
}

func (m *GoBGPPerPeerMetric) SetVpnv4UpdateRcvd(value float64) {
	m.Vpnv4RouteRcvd = value
}

func (m *GoBGPPerPeerMetric) SetVpnv4UpdateSent(value float64) {
	m.Vpnv4RouteAdvd = value
}

// GoBGPPerPeerMetrics contains GoBGP BGP advertised/received routes counts for BGP peer
var GoBGPPerPeerMetrics = make(map[string]*GoBGPPerPeerMetric)

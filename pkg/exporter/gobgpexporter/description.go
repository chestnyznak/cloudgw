package gobgpexporter

import "github.com/prometheus/client_golang/prometheus"

var (
	gobgpVRFCount = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "vrf", "total"),
		"Number of GoBGP VRF",
		nil,
		nil,
	)
	gobgpPeerCount = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "peer", "total"),
		"Number of GoBGP peer",
		nil,
		nil,
	)
	gobgpActivePeerCount = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "active_peer", "total"),
		"Number of GoBGP active peer",
		nil,
		nil,
	)

	gobgpIPv4RouteRcvd = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "ipv4_route_received", "total"),
		"Number of received IPv4 routes",
		[]string{"peer"},
		nil,
	)
	gobgpIPv4RouteAdvd = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "ipv4_route_advertised", "total"),
		"Number of advertized IPv4 routes",
		[]string{"peer"},
		nil,
	)
	gobgpVpnv4RouteRcvd = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "vpnv4_route_received", "total"),
		"Number of received VPNv4 routes",
		[]string{"peer"},
		nil,
	)
	gobgpVpnv4RouteAdvd = prometheus.NewDesc(
		prometheus.BuildFQName("gobgp", "vpnv4_route_advertised", "total"),
		"Number of advertized VPNv4 routes",
		[]string{"peer"},
		nil,
	)
)

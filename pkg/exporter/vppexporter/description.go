package vppexporter

import "github.com/prometheus/client_golang/prometheus"

var (
	vppIPv4RouteTotal = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "ipv4_route", "total"),
		"Number of ipv4 routes in the vrf",
		[]string{"table"},
		nil,
	)
	vppFIPRouteTotal = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "floating_ip_route", "total"),
		"Number of floating ip routes in the vrf",
		[]string{"table"},
		nil,
	)
	vppUDPTunnelTotal = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "udp_tunnel", "total"),
		"Number of udp tunnels",
		[]string{"table"},
		nil,
	)
	// vpp_network_... absolute values

	vppNetworkInterfaceID = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "interface_id"),
		"vpp_network_interface_id interface_id value",
		[]string{"interface"},
		nil,
	)
	vppNetworkRxPacketCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "rx_packet_count"),
		"vpp_network_rx_packet_count Network device statistic rx_packets",
		[]string{"interface"},
		nil,
	)
	vppNetworkRxByteCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "rx_byte_count"),
		"vpp_network_rx_byte_count Network device statistic rx_bytes",
		[]string{"interface"},
		nil,
	)
	vppNetworkRxErrorCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "rx_error_count"),
		"vpp_network_rx_error_count Network device statistic rx_errs",
		[]string{"interface"},
		nil,
	)
	vppNetworkTxPacketCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "tx_packet_count"),
		"vpp_network_tx_packet_count Network device statistic tx_packets",
		[]string{"interface"},
		nil,
	)
	vppNetworkTxByteCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "tx_byte_count"),
		"vpp_network_tx_byte_count Network device statistic tx_bytes",
		[]string{"interface"},
		nil,
	)
	vppNetworkTxErrorCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "tx_error_count"),
		"vpp_network_tx_error_count Network device statistic tx_errs",
		[]string{"interface"},
		nil,
	)
	vppNetworkDropCount = prometheus.NewDesc(
		prometheus.BuildFQName("vpp", "network", "drop_count"),
		"vpp_network_drop_count Network device statistic drop",
		[]string{"interface"},
		nil,
	)
)

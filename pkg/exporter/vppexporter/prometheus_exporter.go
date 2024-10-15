package vppexporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

type CloudgwExporter struct{}

func NewCloudgwExporter() *CloudgwExporter {
	return &CloudgwExporter{}
}

func (c *CloudgwExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- vppIPv4RouteTotal
	ch <- vppFIPRouteTotal
	ch <- vppUDPTunnelTotal
	ch <- vppNetworkInterfaceID
	ch <- vppNetworkRxPacketCount
	ch <- vppNetworkRxByteCount
	ch <- vppNetworkRxErrorCount
	ch <- vppNetworkTxPacketCount
	ch <- vppNetworkTxByteCount
	ch <- vppNetworkTxErrorCount
	ch <- vppNetworkDropCount
}

func (c *CloudgwExporter) Collect(metricsCh chan<- prometheus.Metric) {
	for _, vrfMetric := range VPPVRFMetrics {
		metricsCh <- prometheus.MustNewConstMetric(
			vppIPv4RouteTotal,
			prometheus.GaugeValue,
			vrfMetric.IPv4RouteTotal,
			vrfMetric.VRFName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppFIPRouteTotal,
			prometheus.GaugeValue,
			vrfMetric.FIPRouteTotal,
			vrfMetric.VRFName,
		)
	}

	metricsCh <- prometheus.MustNewConstMetric(
		vppUDPTunnelTotal,
		prometheus.GaugeValue,
		VPPUDPTunnelMetrics.UDPTunnelTotal,
		"default",
	)

	for _, i := range VPPInterfaceMetrics {
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkInterfaceID,
			prometheus.GaugeValue,
			i.interfaceID,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkRxPacketCount,
			prometheus.GaugeValue,
			i.rxPackets,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkRxByteCount,
			prometheus.GaugeValue,
			i.rxBytes,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkRxErrorCount,
			prometheus.GaugeValue,
			i.rxErrors,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkTxPacketCount,
			prometheus.GaugeValue,
			i.txPackets,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkTxByteCount,
			prometheus.GaugeValue,
			i.txBytes,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkTxErrorCount,
			prometheus.GaugeValue,
			i.txErrors,
			i.interfaceName,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			vppNetworkDropCount,
			prometheus.GaugeValue,
			i.dropPackets,
			i.interfaceName,
		)
	}
}

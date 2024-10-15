package gobgpexporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

type CloudgwExporter struct{}

func NewCloudgwExporter() *CloudgwExporter {
	return &CloudgwExporter{}
}

func (c *CloudgwExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- gobgpVRFCount
	ch <- gobgpPeerCount
	ch <- gobgpActivePeerCount
	ch <- gobgpIPv4RouteRcvd
	ch <- gobgpIPv4RouteAdvd
	ch <- gobgpVpnv4RouteRcvd
	ch <- gobgpVpnv4RouteAdvd
}

func (c *CloudgwExporter) Collect(metricsCh chan<- prometheus.Metric) {
	metricsCh <- prometheus.MustNewConstMetric(
		gobgpVRFCount,
		prometheus.CounterValue,
		float64(GoBGPGeneralMetrics.VRFCount),
	)
	metricsCh <- prometheus.MustNewConstMetric(
		gobgpPeerCount,
		prometheus.CounterValue,
		float64(GoBGPGeneralMetrics.PeerCount),
	)
	metricsCh <- prometheus.MustNewConstMetric(
		gobgpActivePeerCount,
		prometheus.CounterValue,
		float64(GoBGPGeneralMetrics.ActivePeerCount),
	)

	for peer, peerMetrics := range GoBGPPerPeerMetrics {
		metricsCh <- prometheus.MustNewConstMetric(
			gobgpIPv4RouteRcvd,
			prometheus.CounterValue,
			peerMetrics.IPv4RouteRcvd,
			peer,
		)

		metricsCh <- prometheus.MustNewConstMetric(
			gobgpIPv4RouteAdvd,
			prometheus.CounterValue,
			peerMetrics.IPv4RouteAdvd,
			peer,
		)

		metricsCh <- prometheus.MustNewConstMetric(
			gobgpVpnv4RouteRcvd,
			prometheus.CounterValue,
			peerMetrics.Vpnv4RouteRcvd,
			peer,
		)
		metricsCh <- prometheus.MustNewConstMetric(
			gobgpVpnv4RouteAdvd,
			prometheus.CounterValue,
			peerMetrics.Vpnv4RouteAdvd,
			peer,
		)
	}
}

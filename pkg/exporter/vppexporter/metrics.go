package vppexporter

// VPPUDPTunnelMetric describes udp tunnel total
type VPPUDPTunnelMetric struct {
	UDPTunnelTotal float64
}

func (m *VPPUDPTunnelMetric) SetUDPTunnelTotal(value float64) {
	m.UDPTunnelTotal = value
}

var VPPUDPTunnelMetrics VPPUDPTunnelMetric

// VPPVRFMetric describes vrf floating ip and ipv4 route total
type VPPVRFMetric struct {
	VRFName        string
	FIPRouteTotal  float64
	IPv4RouteTotal float64
}

func NewVPPVRFMetric(vrfName string) *VPPVRFMetric {
	return &VPPVRFMetric{
		VRFName: vrfName,
	}
}

func (m *VPPVRFMetric) SetVPPRouteTotal(ipRouteNum, fipRouteNum float64) {
	m.FIPRouteTotal = fipRouteNum
	m.IPv4RouteTotal = ipRouteNum
}

var VPPVRFMetrics = make(map[uint32]*VPPVRFMetric)

// VPPInterfaceMetric describes vpp interface metrics for all vrfs
type VPPInterfaceMetric struct {
	interfaceName string
	interfaceID   float64
	rxPackets     float64
	rxBytes       float64
	rxErrors      float64
	txPackets     float64
	txBytes       float64
	txErrors      float64
	dropPackets   float64
}

func NewVPPInterfaceMetric(interfaceName string, interfaceID float64) *VPPInterfaceMetric {
	return &VPPInterfaceMetric{
		interfaceName: interfaceName,
		interfaceID:   interfaceID,
	}
}

func (m *VPPInterfaceMetric) SetVPPInterfaceMetric(
	rxPackets, rxBytes, rxErrors, txPackets, txBytes, txErrors, dropPackets float64,
) {
	m.rxPackets = rxPackets
	m.rxBytes = rxBytes
	m.rxErrors = rxErrors
	m.txPackets = txPackets
	m.txBytes = txBytes
	m.txErrors = txErrors
	m.dropPackets = dropPackets
}

var VPPInterfaceMetrics = make(map[uint32]*VPPInterfaceMetric)

package model

import (
	"math"
	"math/rand/v2"
)

type VPPUDPTunnel struct {
	RoutingTableID uint32 // always = 0 as global routing table has id = 0
	TunnelID       uint32
	SrcIP          string // e.g. "203.0.113.1"
	DstIP          string // e/g. "203.0.113.254"
	SrcPort        uint16 // random 49152 to 65535 (https://datatracker.ietf.org/doc/rfc7510/)
	DstPort        uint16 // 6635 by rfc7510
	FIPServed      uint32
}

const (
	UndefinedTunnelID uint32 = math.MaxUint32
)

func RandUDPTunnelSrcPort() uint16 {
	return uint16(rand.UintN(65535-49152+1) + 49152)
}

func NewVPPUDPTunnel(
	tunnelID uint32,
	srcIP string,
	dstIP string,
	srcPort uint16,
) VPPUDPTunnel {
	return VPPUDPTunnel{
		RoutingTableID: 0, // always default routing table with id = 0
		TunnelID:       tunnelID,
		SrcIP:          srcIP,
		DstIP:          dstIP,
		SrcPort:        srcPort,
		DstPort:        uint16(6635),
		FIPServed:      0,
	}
}

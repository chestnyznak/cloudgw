package model

import (
	bgpapi "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type BGPVRFTable struct {
	Name     string
	ID       uint32 // 0 - global routing table; 1, 2, 3... - VRF ID of customer VRF
	LocalASN uint32
	PeerASN  uint32
	RD       *anypb.Any   // cloudgwRID:vrfID
	ExportRT []*anypb.Any // cloudgwASN:vrfID
	ImportRT []*anypb.Any // tfASN:vrfID
}

func NewBGPVRFTable(
	name string,
	id uint32,
	localASN uint32,
	peerASN uint32,
	rd *anypb.Any,
	exportRT []*anypb.Any,
	importRT []*anypb.Any,
) BGPVRFTable {
	return BGPVRFTable{
		Name:     name,
		ID:       id,
		LocalASN: localASN,
		PeerASN:  peerASN,
		RD:       rd,
		ExportRT: exportRT,
		ImportRT: importRT,
	}
}

// RD returns RD for CloudGW's VRF (routerID:vrfID)
func RD(routerID string, vrfID uint32) *anypb.Any {
	rd, _ := anypb.New(&bgpapi.RouteDistinguisherIPAddress{
		Admin:    routerID,
		Assigned: vrfID,
	})

	return rd
}

// RT returns RT in TwoOctetAsSpecific format (ASN:vrfID)
func RT(asn uint32, vrfID uint32) *anypb.Any {
	rt, _ := anypb.New(&bgpapi.TwoOctetAsSpecificExtended{
		IsTransitive: true,
		SubType:      2,
		Asn:          asn,
		LocalAdmin:   vrfID,
	})

	return rt
}

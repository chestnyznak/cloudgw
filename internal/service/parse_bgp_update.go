package service

import (
	bgpapi "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/encoding/protojson"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
)

// ParseBGPUpdate parses BGP IPv4/VPNv4 Update and returns VPPIPRoute struct (all fields except RD/RT) with type flags
func ParseBGPUpdate(
	bgpPath *bgpapi.Path,
	vppVRFIDToNHMap map[uint32]string,
	bgpPeerToPeerTypeMap map[string]int,
	tfASN uint32,
	cloudgwASN uint32,
) (
	fromTF bool,
	fromPN bool,
	bgpNLRIAttrs gobgpapi.BGPNLRIAttrs,
	err error,
) {
	marshaller := protojson.MarshalOptions{
		Indent:        "  ",
		UseProtoNames: true,
	}

	// skip internal updates with peer = 0.0.0.0 as useless
	if _, ok := bgpPeerToPeerTypeMap[bgpPath.NeighborIp]; !ok {
		return false, false, bgpNLRIAttrs, nil
	}

	nlri, err := marshaller.Marshal(bgpPath.Nlri)
	if err != nil {
		return false, false, bgpNLRIAttrs, err
	}

	peerType, ok := bgpPeerToPeerTypeMap[bgpPath.NeighborIp]
	if !ok {
		return false, false, bgpNLRIAttrs, nil
	}

	switch peerType {
	case model.TF:
		pathAttrsCommunities, err := marshaller.Marshal(bgpPath.Pattrs[3])
		if err != nil {
			return false, false, bgpNLRIAttrs, err
		}

		return ParseVPNv4UpdateFromTF(nlri, vppVRFIDToNHMap, tfASN, len(bgpPath.Pattrs), pathAttrsCommunities)

	case model.PHYNET:
		return ParseVPNv4UpdateFromPHYNET(nlri, vppVRFIDToNHMap, cloudgwASN, bgpPath.SourceAsn, bgpPath.NeighborIp)
	}

	return false, false, bgpNLRIAttrs, nil
}

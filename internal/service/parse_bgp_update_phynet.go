package service

import (
	"github.com/tidwall/gjson"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
)

func ParseVPNv4UpdateFromPHYNET(
	nlri []byte,
	vppVRFIDToNHMap map[uint32]string,
	cloudgwASN uint32,
	nlriSourceASN uint32, // bgpPath.SourceAsn, where bgpPath *bgpapi.Path
	nlriNeighborIP string, // bgpPath.NeighborIp, where bgpPath *bgpapi.Path
) (
	fromTF bool,
	fromPN bool,
	bgpNLRIAttrs gobgpapi.BGPNLRIAttrs,
	err error,
) {
	var (
		parsedPrefixAddr gjson.Result
		parsedPrefixLen  gjson.Result
		parsedVRF        uint32
	)

	if nlriSourceASN != cloudgwASN { // ignore own route from physical network
		parsedPrefixAddr = gjson.Get(string(nlri), "prefix")
		parsedPrefixLen = gjson.Get(string(nlri), "prefix_len")

		if parsedPrefixLen.String() == "" {
			parsedPrefixLen = gjson.Parse("0") // as nlri with default route may not have prefix_len
		}

		// find vrf by next-hop
		for v, nh := range vppVRFIDToNHMap {
			if nlriNeighborIP == nh {
				parsedVRF = v

				break
			}
		}

		bgpNLRIAttrs = gobgpapi.NewBGPNLRIAttrs(
			parsedPrefixAddr.String()+"/"+parsedPrefixLen.String(),
			nlriNeighborIP, // for physical network BGP neighbor ip and bgp nexthop are the same
			parsedVRF,
			nil,
			nil,
			[]uint32{model.UndefinedLabel},
		)
	}

	fromPN = true
	fromTF = false

	return fromTF, fromPN, bgpNLRIAttrs, nil
}

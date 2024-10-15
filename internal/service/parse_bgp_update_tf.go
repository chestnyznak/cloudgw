package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
)

func ParseVPNv4UpdateFromTF(
	nlri []byte,
	vppVRFIDToNHMap map[uint32]string,
	tfASN uint32,
	pattrsLen int,
	pattrsCommunities []byte,
) (
	fromTF bool,
	fromPN bool,
	bgpNLRIAttrs gobgpapi.BGPNLRIAttrs,
	err error,
) {
	var (
		parsedPrefixAddr   gjson.Result
		parsedPrefixLen    gjson.Result
		parsedRDAdmin      gjson.Result
		parsedFipMPLSLabel gjson.Result
		parsedCommunities  gjson.Result
	)

	parsedPrefixAddr = gjson.Get(string(nlri), "prefix")
	parsedPrefixLen = gjson.Get(string(nlri), "prefix_len")

	if parsedPrefixLen.String() == "" {
		parsedPrefixLen = gjson.Parse("0") // as nlri with default route may not have prefix_len
	}

	parsedRDAdmin = gjson.Get(string(nlri), "rd.admin")
	parsedFipMPLSLabel = gjson.Get(string(nlri), "labels.0")

	// parse update with MP_REACH_NLRI/MP_UNREACH_NLRI from tungsten fabric

	if pattrsLen < 4 {
		return false, false, bgpNLRIAttrs, fmt.Errorf("wrong bgp update nlri pattrs length")
	}

	parsedCommunities = gjson.Get(string(pattrsCommunities), "communities")
	communitiesArray := parsedCommunities.Array()

	// find vrf by rt/rd from communities

	for _, communities := range communitiesArray {
		parsedASN := communities.Get("asn")

		if parsedASN.Exists() {
			asn, err := strconv.Atoi(parsedASN.String())
			if err != nil {
				return false, false, bgpNLRIAttrs, err
			}

			parsedLocalAdmin := communities.Get("local_admin")

			localAdmin, err := strconv.Atoi(parsedLocalAdmin.String())
			if err != nil {
				return false, false, bgpNLRIAttrs, err
			}

			// ATTENTION: hardcoded algorithm of rt matching

			if asn == int(tfASN) {
				if _, ok := vppVRFIDToNHMap[uint32(localAdmin)]; ok {
					if pattrsLen < 5 {
						return false, false, bgpNLRIAttrs, fmt.Errorf("wrong bgp update nlri pattrs length")
					}

					bgpNLRIAttrs = gobgpapi.NewBGPNLRIAttrs(
						strings.Join([]string{parsedPrefixAddr.String(), parsedPrefixLen.String()}, "/"),
						parsedRDAdmin.String(),
						uint32(localAdmin),
						nil,
						nil,
						[]uint32{uint32(parsedFipMPLSLabel.Int())}, // label = 0 for withdraw
					)

					fromPN = false
					fromTF = true

					return fromTF, fromPN, bgpNLRIAttrs, err
				}
			}
		}
	}

	return fromTF, fromPN, bgpNLRIAttrs, err
}

package model

import (
	"strconv"
	"strings"

	"go.fd.io/govpp/binapi/interface_types"
)

// VPPIPRoute contains a vpp floating ip or ipv4 route (short-lived structure used for adding/deleting and dumping ip routes)
type VPPIPRoute struct {
	VRFID           uint32
	MainInterfaceID interface_types.InterfaceIndex
	SubInterfaceID  interface_types.InterfaceIndex
	Prefix          string   // e.g. "203.0.113.0/24"
	NextHops        []string // e.g. ["203.0.113.254", "203.0.114.254"]
	TunnelIDs       []uint32
	FIPMPLSLabels   []uint32
}

func NewVPPIPRoute(
	vrfID uint32,
	mainInterfaceID interface_types.InterfaceIndex,
	subInterfaceID interface_types.InterfaceIndex,
	prefix string,
	nextHops []string,
	tunnelIDs []uint32,
	fipMPLSLabels []uint32,
) VPPIPRoute {
	return VPPIPRoute{
		VRFID:           vrfID,
		MainInterfaceID: mainInterfaceID,
		SubInterfaceID:  subInterfaceID,
		Prefix:          prefix,
		NextHops:        nextHops,
		TunnelIDs:       tunnelIDs,
		FIPMPLSLabels:   fipMPLSLabels,
	}
}

func (r *VPPIPRoute) MPLSLabels() string {
	if len(r.FIPMPLSLabels) == 0 {
		return ""
	}

	labels := make([]string, len(r.FIPMPLSLabels))

	for i := range r.FIPMPLSLabels {
		labels[i] = strconv.Itoa(int(r.FIPMPLSLabels[i]))
	}

	return strings.Join(labels, ",")
}

// AddPath adds a new path to the VPPIPRoute. If the path with specific next-hop already exists, it is replaced with new one
func (r *VPPIPRoute) AddPath(nextHop string, tunnelID, mplsLabel uint32) {
	if nextHop == "" || tunnelID == 0 || mplsLabel == 0 {
		return
	}

	if len(r.NextHops) != 0 {
		for i := range r.NextHops {
			if r.NextHops[i] == nextHop {
				r.NextHops = append(r.NextHops[:i], r.NextHops[i+1:]...)
				r.TunnelIDs = append(r.TunnelIDs[:i], r.TunnelIDs[i+1:]...)
				r.FIPMPLSLabels = append(r.FIPMPLSLabels[:i], r.FIPMPLSLabels[i+1:]...)
			}
		}
	}

	r.NextHops = append(r.NextHops, nextHop)
	r.TunnelIDs = append(r.TunnelIDs, tunnelID)
	r.FIPMPLSLabels = append(r.FIPMPLSLabels, mplsLabel)
}

// DelPath deletes a path from the VPPIPRoute based on netxhop
func (r *VPPIPRoute) DelPath(nextHop string) {
	if len(r.NextHops) == 1 && r.NextHops[0] == nextHop {
		r.NextHops = nil
		r.TunnelIDs = nil
		r.FIPMPLSLabels = nil

		return
	}

	for i := range r.NextHops {
		if r.NextHops[i] == nextHop {
			r.NextHops = append(r.NextHops[:i], r.NextHops[i+1:]...)
			r.TunnelIDs = append(r.TunnelIDs[:i], r.TunnelIDs[i+1:]...)
			r.FIPMPLSLabels = append(r.FIPMPLSLabels[:i], r.FIPMPLSLabels[i+1:]...)

			return
		}
	}
}

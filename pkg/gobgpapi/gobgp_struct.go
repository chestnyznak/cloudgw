package gobgpapi

import (
	"time"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)

// BGPNLRIAttrs contains BGP NLRI attributes. Used for advertise/withdraw/parse BGP prefix to/from BGP tables
type BGPNLRIAttrs struct {
	Prefix    string // e.g. "203.0.113.0/24"
	NextHop   string // e.g. "203.0.113.1"
	VRFID     uint32
	AFI       bgpapi.Family_Afi
	SAFI      bgpapi.Family_Safi
	RD        *anypb.Any
	RT        []*anypb.Any
	MPLSLabel []uint32
}

func NewBGPNLRIAttrs(
	prefix string,
	nextHop string,
	vrfID uint32,
	rd *anypb.Any,
	rt []*anypb.Any,
	mplsLabel []uint32,
) BGPNLRIAttrs {
	return BGPNLRIAttrs{
		Prefix:    prefix,
		NextHop:   nextHop,
		VRFID:     vrfID,
		AFI:       bgpapi.Family_AFI_IP,
		SAFI:      bgpapi.Family_SAFI_MPLS_VPN,
		RD:        rd,
		RT:        rt,
		MPLSLabel: mplsLabel,
	}
}

type BGPVRFTable struct {
	Name     string
	ID       uint32 // 0 - global routing table, 1, 2, 3... - vrf id of customer VRF
	ASN      uint32
	PeerASN  uint32
	RD       *anypb.Any   // cloudgwRID:vrfID
	ExportRT []*anypb.Any // cloudgwASN:vrfID
	ImportRT []*anypb.Any // tfASN:vrfID
}

func NewBGPVRFTable(
	name string,
	id uint32,
	asn uint32,
	peerASN uint32,
	rd *anypb.Any,
	exportRT []*anypb.Any,
	importRT []*anypb.Any,
) BGPVRFTable {
	return BGPVRFTable{
		Name:     name,
		ID:       id,
		ASN:      asn,
		PeerASN:  peerASN,
		RD:       rd,
		ExportRT: exportRT,
		ImportRT: importRT,
	}
}

type BGPPeer struct {
	PeerType            int // TF or PHYNET
	PeerASN             uint32
	PeerAddress         string // e.g. "203.0.113.1"
	PeerPort            uint32 // e.g. 179
	Md5Password         string
	EbgpMultiHop        bool
	EbgpMultiHopTTL     uint32
	VRFName             string
	AFI                 bgpapi.Family_Afi
	SAFI                bgpapi.Family_Safi
	KeepAliveTimer      uint64
	HoldTimer           uint64
	BGPPeerState        bgpapi.PeerState_SessionState
	BGPPeerPrevState    bgpapi.PeerState_SessionState
	BGPPeerLastActivity time.Time
}

const (
	TF     int = 0
	PHYNET int = 1
)

func NewBGPPeer(
	peerType int, // model.TF / model.PHYNET
	peerASN uint32,
	peerAddress string,
	peerPort uint32,
	md5Password string,
	ebgpMultiHop bool,
	ebgpMultiHopTTL uint32,
	vrfName string,
	keepAliveTimer uint64,
	holdTimer uint64,
) BGPPeer {
	var peer BGPPeer

	switch peerType {
	case TF:
		peer = BGPPeer{
			PeerType:            peerType,
			PeerASN:             peerASN,
			PeerAddress:         peerAddress,
			PeerPort:            peerPort,
			Md5Password:         md5Password,
			EbgpMultiHop:        ebgpMultiHop,
			EbgpMultiHopTTL:     ebgpMultiHopTTL,
			VRFName:             vrfName,
			AFI:                 bgpapi.Family_AFI_IP,
			SAFI:                bgpapi.Family_SAFI_MPLS_VPN,
			KeepAliveTimer:      keepAliveTimer,
			HoldTimer:           holdTimer,
			BGPPeerState:        bgpapi.PeerState_UNKNOWN,
			BGPPeerPrevState:    bgpapi.PeerState_UNKNOWN,
			BGPPeerLastActivity: time.Now(),
		}
	case PHYNET:
		peer = BGPPeer{
			PeerType:            peerType,
			PeerASN:             peerASN,
			PeerAddress:         peerAddress,
			PeerPort:            peerPort,
			Md5Password:         md5Password,
			EbgpMultiHop:        ebgpMultiHop,
			EbgpMultiHopTTL:     ebgpMultiHopTTL,
			VRFName:             vrfName,
			AFI:                 bgpapi.Family_AFI_IP,
			SAFI:                bgpapi.Family_SAFI_UNICAST,
			KeepAliveTimer:      keepAliveTimer,
			HoldTimer:           holdTimer,
			BGPPeerState:        bgpapi.PeerState_UNKNOWN,
			BGPPeerPrevState:    bgpapi.PeerState_UNKNOWN,
			BGPPeerLastActivity: time.Now(),
		}
	}

	return peer
}

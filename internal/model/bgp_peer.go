package model

import (
	"time"

	bgpapi "github.com/osrg/gobgp/v3/api"
)

const (
	TF     int = 0
	PHYNET int = 1
)

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
	BFDPeering          *BFDPeer // nil for tungsten fabric controllers
}

type BFDPeer struct {
	BFDEnabled         bool
	BFDPeerIP          string // e.g. "203.0.113.1"
	BFDPeerEstablished bool
	BFDLocalIP         string // e.g. "203.0.113.2"
	BFDTxRate          int
	BFDRxMin           int
	BFDMultiplier      int
}

func NewBGPPeer(
	peerType int, // domain.TF / domain.PHYNET
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
			BFDPeering:          nil,
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
			BFDPeering:          nil,
		}
	}

	return peer
}

func NewBFDPeer(
	bfdEnabled bool,
	bfdPeerIP string,
	bfdLocalIP string,
	bfdTxRate int,
	bfdRxMin int,
	bfdMultiplier int,
) BFDPeer {
	return BFDPeer{
		BFDEnabled:         bfdEnabled,
		BFDPeerIP:          bfdPeerIP,
		BFDPeerEstablished: false,
		BFDLocalIP:         bfdLocalIP,
		BFDTxRate:          bfdTxRate,
		BFDRxMin:           bfdRxMin,
		BFDMultiplier:      bfdMultiplier,
	}
}

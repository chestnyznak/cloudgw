package model

import (
	"math"

	"go.fd.io/govpp/binapi/interface_types"
)

// VPPVRFTable is VPP Routing Table (Global Routing Table and VRFs)
// NOTE: NextHop is Default GW for GRT, and BGP Peer for VRFs, hardcoded.
// NOTE: MainInterfaceID always 1 for now (0 loopback Interface ID)
type VPPVRFTable struct {
	Name            string
	ID              uint32
	MainInterfaceID interface_types.InterfaceIndex
	SubInterfaceID  interface_types.InterfaceIndex
	VLAN            uint32
	LocalAddr       string // e.g. "203.0.113.1"
	LocalAddrLen    uint32 // e.g. 24
	NextHop         string // e.g. "203.0.113.254"
	MPLSLocalLabel  uint32
	FIPPrefixes     []string
	FIPServed       uint32
}

const (
	UndefinedSubIf  interface_types.InterfaceIndex = math.MaxUint32
	UndefinedMainIf interface_types.InterfaceIndex = math.MaxUint32
	UndefinedLabel  uint32                         = 0
)

func NewVPPVRFTable(
	name string,
	id uint32,
	mainInterfaceID interface_types.InterfaceIndex,
	subInterfaceID interface_types.InterfaceIndex,
	vlan uint32,
	localAddr string,
	localAddrLen uint32,
	nextHop string,
	mplsLocalLabel uint32,
	fipPrefixes []string,
) VPPVRFTable {
	return VPPVRFTable{
		Name:            name,
		ID:              id,
		MainInterfaceID: mainInterfaceID,
		SubInterfaceID:  subInterfaceID,
		VLAN:            vlan,
		LocalAddr:       localAddr,
		LocalAddrLen:    localAddrLen,
		NextHop:         nextHop,
		MPLSLocalLabel:  mplsLocalLabel,
		FIPPrefixes:     fipPrefixes,
		FIPServed:       0,
	}
}

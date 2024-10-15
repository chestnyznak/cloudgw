package vpp

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.fd.io/govpp"
	"go.fd.io/govpp/adapter/socketclient"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/fib_types"
	interfaces "go.fd.io/govpp/binapi/interface"
	"go.fd.io/govpp/binapi/interface_types"
	"go.fd.io/govpp/binapi/ip"
	"go.fd.io/govpp/binapi/ip_types"
	"go.fd.io/govpp/binapi/memclnt"
	"go.fd.io/govpp/binapi/mpls"
	"go.fd.io/govpp/binapi/udp"
	"go.fd.io/govpp/binapi/vpe"
	"go.fd.io/govpp/core"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

// ConnectToVPPAPIAsync connects to VPP asynchronously
func ConnectToVPPAPIAsync(ctx context.Context, sockAddr string) (api.Stream, func(), chan core.ConnectionEvent, error) {
	if sockAddr == "" {
		sockAddr = socketclient.DefaultSocketName
	}

	conn, connEvent, err := govpp.AsyncConnect(sockAddr, core.DefaultMaxReconnectAttempts, core.DefaultReconnectInterval)
	if err != nil {
		return nil, nil, connEvent, fmt.Errorf("failed to initialize connection to vpp: %w", err)
	}

	// wait for connected event

	event := <-connEvent
	if event.State != core.Connected {
		return nil, nil, connEvent, fmt.Errorf("failed to connect to vpp: %w", err)
	}

	// check compatibility of used messages

	ch, err := conn.NewAPIChannel()
	if err != nil {
		return nil, nil, connEvent, fmt.Errorf("failed to create new api channel: %w", err)
	}

	if err := ch.CheckCompatiblity(vpe.AllMessages()...); err != nil {
		return nil, nil, connEvent, fmt.Errorf("vpp channel compatibility check failed: %w", err)
	}

	if err := ch.CheckCompatiblity(interfaces.AllMessages()...); err != nil {
		return nil, nil, connEvent, fmt.Errorf("vpp channel compatibility check failed: %w", err)
	}

	stream, err := conn.NewStream(
		ctx,
		core.WithRequestSize(50),
		core.WithReplySize(50),
		core.WithReplyTimeout(2*time.Second))
	if err != nil {
		return nil, nil, connEvent, fmt.Errorf("failed to create new stream: %w", err)
	}

	return stream, conn.Disconnect, connEvent, nil
}

// GetVPPVersion gets and returns VPP version
func GetVPPVersion(stream api.Stream) (string, error) {
	req := &vpe.ShowVersion{}

	if err := stream.SendMsg(req); err != nil {
		return "", err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return "", err
	}

	reply := msg.(*vpe.ShowVersionReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return "", api.RetvalToVPPApiError(reply.Retval)
	}

	return reply.Version, nil
}

// AddDelMPLSTable adds/deletes MPLS LFIB table 0 (mpls table add|del 0)
func AddDelMPLSTable(stream api.Stream, isAdd bool) error {
	req := &mpls.MplsTableAddDel{
		MtIsAdd: isAdd,
		MtTable: mpls.MplsTable{
			MtTableID: 0,
			MtName:    "default",
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*mpls.MplsTableAddDelReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// GetInterfaceIDs gets IDs of all VPP Interfaces
func GetInterfaceIDs(stream api.Stream) ([]interface_types.InterfaceIndex, error) {
	req := &interfaces.SwInterfaceDump{}

	if err := stream.SendMsg(req); err != nil {
		return nil, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return nil, err
	}

	var interfaceIDs []interface_types.InterfaceIndex

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return nil, err
		}

		switch reply := msg.(type) {
		case *interfaces.SwInterfaceDetails:
			interfaceIDs = append(interfaceIDs, reply.SwIfIndex)

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return nil, fmt.Errorf("unexpected message type: %T", msg)
		}
	}

	return interfaceIDs, nil
}

// SetupMainInterface create a configuration VPP main interface (interfaceID=1 in most case as loopback interface has interfaceID=0).
// NOTE: Used as source of MPLS over UDP tunnels to vRouters.
func SetupMainInterface(stream api.Stream, vppVRFTable model.VPPVRFTable) error {
	interfaceIndex := vppVRFTable.MainInterfaceID

	// add ip address (set interface ip address add <interface> <ip-address>)

	{
		interfacePrefix, err := ip_types.ParseAddressWithPrefix(
			vppVRFTable.LocalAddr + "/" + strconv.FormatUint(uint64(vppVRFTable.LocalAddrLen), 10))
		if err != nil {
			return err
		}

		req := &interfaces.SwInterfaceAddDelAddress{
			SwIfIndex: interfaceIndex,
			IsAdd:     true,
			Prefix:    interfacePrefix,
		}

		if err := stream.SendMsg(req); err != nil {
			return err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return err
		}

		reply := msg.(*interfaces.SwInterfaceAddDelAddressReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("ip address of main interface is configured", "address", vppVRFTable.LocalAddr)
	}

	// enable the main interface (set interface state <interface> enable)

	{
		req := &interfaces.SwInterfaceSetFlags{
			SwIfIndex: interfaceIndex,
			Flags:     interface_types.IF_STATUS_API_FLAG_ADMIN_UP,
		}

		if err := stream.SendMsg(req); err != nil {
			return err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return err
		}

		reply := msg.(*interfaces.SwInterfaceSetFlagsReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("main interface enabled", "interface id", vppVRFTable.MainInterfaceID)
	}

	// enable mpls on the main interface (set interface mpls <interface> enable)

	{
		req := &mpls.SwInterfaceSetMplsEnable{
			SwIfIndex: interfaceIndex,
			Enable:    true,
		}

		if err := stream.SendMsg(req); err != nil {
			return err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return err
		}

		reply := msg.(*mpls.SwInterfaceSetMplsEnableReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("mpls enabled on main interface", "interface id", vppVRFTable.MainInterfaceID)
	}

	return nil
}

// ResetMainInterface resets VPP main interface (delete IP address and disable the interface).
// NOTE: interfaceID=1 in most cases as loopback interface has interfaceID=0.
func ResetMainInterface(stream api.Stream, mainInterfaceID uint32) error {
	swIfIndex := interface_types.InterfaceIndex(mainInterfaceID)

	// delete ip address

	{
		req := &interfaces.SwInterfaceAddDelAddress{
			SwIfIndex: swIfIndex,
			IsAdd:     false,
			DelAll:    true,
		}

		if err := stream.SendMsg(req); err != nil {
			return err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return err
		}

		reply := msg.(*interfaces.SwInterfaceAddDelAddressReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("ip address of main interface is deleted", "interface id", mainInterfaceID)
	}

	// disable the interface (sometimes doesn't work, don't care!)

	{
		req := &interfaces.SwInterfaceSetFlags{
			SwIfIndex: swIfIndex,
			Flags:     math.MaxUint32,
		}

		if err := stream.SendMsg(req); err != nil {
			return err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return err
		}

		reply := msg.(*interfaces.SwInterfaceSetFlagsReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("main interface disabled", "interface id", mainInterfaceID)
	}

	return nil
}

// AddDelVRF adds/deletes VPP VRF Table
func AddDelVRF(stream api.Stream, isAdd bool, vppVRFTable model.VPPVRFTable) error {
	req := &ip.IPTableAddDel{
		IsAdd: isAdd,
		Table: ip.IPTable{
			TableID: vppVRFTable.ID,
			Name:    vppVRFTable.Name,
			IsIP6:   false,
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*ip.IPTableAddDelReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// AddSubInterface configures a VPP sub-interface used for physical network connection for specific VRF/VLAN.
func AddSubInterface(stream api.Stream, vppVRFTable *model.VPPVRFTable) (interface_types.InterfaceIndex, error) {
	mainInterfaceID := vppVRFTable.MainInterfaceID

	var subInterfaceID interface_types.InterfaceIndex

	// create a sub-interface

	{
		req := &interfaces.CreateVlanSubif{
			SwIfIndex: mainInterfaceID,
			VlanID:    vppVRFTable.VLAN,
		}

		if err := stream.SendMsg(req); err != nil {
			return model.UndefinedSubIf, err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return model.UndefinedSubIf, err
		}

		reply := msg.(*interfaces.CreateVlanSubifReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return model.UndefinedSubIf, api.RetvalToVPPApiError(reply.Retval)
		}

		subInterfaceID = reply.SwIfIndex

		logger.Debug("sub-interface created", "vlan", vppVRFTable.VLAN)
	}

	// bind the sub-interface to vrf

	{
		req := &interfaces.SwInterfaceSetTable{
			SwIfIndex: subInterfaceID,
			IsIPv6:    false,
			VrfID:     vppVRFTable.ID,
		}

		if err := stream.SendMsg(req); err != nil {
			return model.UndefinedSubIf, err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return model.UndefinedSubIf, err
		}

		reply := msg.(*interfaces.SwInterfaceSetTableReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return model.UndefinedSubIf, api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("sub-interface bound to vrf", "vlan", vppVRFTable.VLAN)
	}

	// set ip address for the sub-interface

	{
		interfaceAddr, _ := ip_types.ParseAddressWithPrefix(
			vppVRFTable.LocalAddr + "/" + strconv.FormatUint(uint64(vppVRFTable.LocalAddrLen), 10),
		)

		req := &interfaces.SwInterfaceAddDelAddress{
			SwIfIndex: subInterfaceID,
			IsAdd:     true,
			Prefix:    interfaceAddr,
		}

		if err := stream.SendMsg(req); err != nil {
			return model.UndefinedSubIf, err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return model.UndefinedSubIf, err
		}

		reply := msg.(*interfaces.SwInterfaceAddDelAddressReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return model.UndefinedSubIf, api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("sub-interface ip address set", "vlan", vppVRFTable.VLAN)
	}

	// enable sub-interface

	{
		req := &interfaces.SwInterfaceSetFlags{
			SwIfIndex: subInterfaceID,
			Flags:     interface_types.IF_STATUS_API_FLAG_ADMIN_UP,
		}

		if err := stream.SendMsg(req); err != nil {
			return model.UndefinedSubIf, err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return model.UndefinedSubIf, err
		}

		reply := msg.(*interfaces.SwInterfaceSetFlagsReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return model.UndefinedSubIf, api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("sub-interface enabled", "vlan", vppVRFTable.VLAN)
	}

	return subInterfaceID, nil
}

// DelSubInterfaces deletes all VPP sub-interfaces of main interfaces for all VRFs (Sub-interface ID > mainInterfaceID) and returns number of deleted sub-interfaces
func DelSubInterfaces(stream api.Stream, mainInterfaceID uint32) (uint32, error) {
	var subInterfaceIDs []interface_types.InterfaceIndex

	removedSubInterfaces := uint32(0)

	req := &interfaces.SwInterfaceDump{}

	if err := stream.SendMsg(req); err != nil {
		return 0, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return 0, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return 0, err
		}

		switch reply := msg.(type) {
		case *interfaces.SwInterfaceDetails:
			if reply.SwIfIndex > interface_types.InterfaceIndex(mainInterfaceID) {
				subInterfaceIDs = append(subInterfaceIDs, reply.SwIfIndex)
			}

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return removedSubInterfaces, fmt.Errorf("unexpected message type: %T", msg)
		}
	}

	if len(subInterfaceIDs) == 0 {
		return 0, nil
	}

	// delete all sub-interfaces

	for _, subInterfaceID := range subInterfaceIDs {
		req := &interfaces.DeleteSubif{
			SwIfIndex: subInterfaceID,
		}

		if err := stream.SendMsg(req); err != nil {
			return removedSubInterfaces, err
		}

		msg, err := stream.RecvMsg()
		if err != nil {
			return removedSubInterfaces, err
		}

		reply := msg.(*interfaces.DeleteSubifReply)

		if api.RetvalToVPPApiError(reply.Retval) != nil {
			return removedSubInterfaces, api.RetvalToVPPApiError(reply.Retval)
		}

		logger.Debug("sub-interface deleted", "sub-interface id", subInterfaceID)

		removedSubInterfaces++
	}

	return removedSubInterfaces, nil
}

// CountUDPTunnels counts all configured UDP tunnels in a VPP (used for metric expose)
func CountUDPTunnels(stream api.Stream) (float64, error) {
	req := &udp.UDPEncapDump{}

	udpTunnelRecords := float64(0)

	if err := stream.SendMsg(req); err != nil {
		return udpTunnelRecords, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return udpTunnelRecords, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return udpTunnelRecords, err
		}

		switch msg.(type) {
		case *udp.UDPEncapDetails:
			udpTunnelRecords++

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return udpTunnelRecords, fmt.Errorf("unexpected message type: %T", msg)
		}
	}

	return udpTunnelRecords, nil
}

// CheckUDPTunnelTableEmpty checks if UDP tunnel table is empty
func CheckUDPTunnelTableEmpty(stream api.Stream) (bool, error) {
	req := &udp.UDPEncapDump{}

	udpTunnelRecords := 0

	if err := stream.SendMsg(req); err != nil {
		return false, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return false, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return false, err
		}

		switch msg.(type) {
		case *udp.UDPEncapDetails:
			udpTunnelRecords++

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return false, fmt.Errorf("unexpected message type: %T", msg)
		}
	}

	if udpTunnelRecords == 0 {
		return true, nil
	}

	return false, nil
}

// DumpUDPTunnels returns all configured UDP tunnels from VPP
func DumpUDPTunnels(stream api.Stream) ([]model.VPPUDPTunnel, error) {
	var (
		dumpedRecords []model.VPPUDPTunnel
		dumpedRecord  model.VPPUDPTunnel
	)

	req := &udp.UDPEncapDump{}

	if err := stream.SendMsg(req); err != nil {
		return nil, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return nil, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return dumpedRecords, err
		}

		switch replay := msg.(type) {
		case *udp.UDPEncapDetails:
			dumpedRecord = model.VPPUDPTunnel{
				TunnelID: replay.UDPEncap.ID,
				SrcIP:    replay.UDPEncap.SrcIP.String(),
				DstIP:    replay.UDPEncap.DstIP.String(),
				SrcPort:  replay.UDPEncap.SrcPort,
			}

			dumpedRecords = append(dumpedRecords, dumpedRecord)

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return dumpedRecords, fmt.Errorf("unexpected message type: %T", msg)
		}
	}

	return dumpedRecords, nil
}

// AddUDPTunnel creates UDP tunnel to specific vRouter/floating IP and fill TunnelID field of VPPUDPTunnelTable struct (udp encap add <vpp_addr> <vrouter> <src_port> 6635 table-id 0)
func AddUDPTunnel(stream api.Stream, vppUDPTunnel *model.VPPUDPTunnel) error {
	srcIP, err := ip_types.ParseAddress(vppUDPTunnel.SrcIP)
	if err != nil {
		return err
	}

	dstIP, err := ip_types.ParseAddress(vppUDPTunnel.DstIP)
	if err != nil {
		return err
	}

	req := &udp.UDPEncapAdd{
		UDPEncap: udp.UDPEncap{
			TableID: vppUDPTunnel.RoutingTableID, // always 0 as mpls inet.0
			SrcIP:   srcIP,
			DstIP:   dstIP,
			SrcPort: vppUDPTunnel.SrcPort,
			DstPort: vppUDPTunnel.DstPort,
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*udp.UDPEncapAddReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	vppUDPTunnel.TunnelID = reply.ID

	return nil
}

// DelUDPTunnel deletes UDP tunnel to specific vRouter/floating IP (udp encap del index <TunnelID>)
func DelUDPTunnel(stream api.Stream, udpTunnelID uint32) error {
	req := &udp.UDPEncapDel{
		ID: udpTunnelID,
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*udp.UDPEncapDelReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// AddUDPDecap registers a port to decapsulate incoming UDP encapsulated packets (port 6635 by RFC7510) (udp decap add 6635 ipv4)
func AddUDPDecap(stream api.Stream) error {
	req := &udp.UDPDecapAddDel{
		IsAdd: true,
		UDPDecap: udp.UDPDecap{
			IsIP4:     1,
			Port:      6635, // rfc7510
			NextProto: udp.UDP_API_DECAP_PROTO_MPLS,
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*udp.UDPDecapAddDelReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// DumpFIPRoutes returns all configured IP/MPLS routes to floating IP addresses for all VRFs (FIB_API_PATH_TYPE_UDP_ENCAP)
func DumpFIPRoutes(stream api.Stream) ([]model.VPPIPRoute, error) {
	var dumpedRouteRecords []model.VPPIPRoute

	// get all ipv4 routing tables (not routes!)

	var dumpedTableRecords []ip.IPTableDetails

	{
		req := &ip.IPTableDump{}

		if err := stream.SendMsg(req); err != nil {
			return nil, err
		}

		if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
			return nil, err
		}

	LoopTable:
		for {
			msg, err := stream.RecvMsg()
			if err != nil {
				return dumpedRouteRecords, err
			}

			switch reply := msg.(type) {
			case *ip.IPTableDetails:
				if !reply.Table.IsIP6 {
					dumpedTableRecords = append(dumpedTableRecords, *reply)
				}

			case *memclnt.ControlPingReply:
				break LoopTable

			default:
				return dumpedRouteRecords, fmt.Errorf("unexpected message type %T", msg)
			}
		}
	}

	// get all ip/mpls floating ip routes

	for _, tableID := range dumpedTableRecords {
		req := &ip.IPRouteV2Dump{
			Table: tableID.Table,
		}

		if err := stream.SendMsg(req); err != nil {
			return nil, err
		}

		if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
			return nil, err
		}

	LoopRoute:
		for {
			msg, err := stream.RecvMsg()
			if err != nil {
				return dumpedRouteRecords, err
			}

			switch reply := msg.(type) {
			case *ip.IPRouteV2Details:
				tunnels := make([]uint32, 0)
				nhs := make([]string, 0)
				labels := make([]uint32, 0)

				for _, p := range reply.Route.Paths {
					tnl := p.Nh.ObjID / 16777216 // div 2^24

					tunnels = append(tunnels, tnl)

					nh := p.Nh.Address.GetIP4().String()

					nhs = append(nhs, nh)

					lbl := p.LabelStack[0].Label

					labels = append(labels, lbl)
				}

				// if first element is tunnel, then other are tunnels also
				if reply.Route.Paths[0].Type.String() == "FIB_API_PATH_TYPE_UDP_ENCAP" { //nolint:goconst
					dumpedRouteRecord := model.NewVPPIPRoute(
						reply.Route.TableID,
						interface_types.InterfaceIndex(reply.Route.Paths[0].SwIfIndex),
						model.UndefinedSubIf,
						reply.Route.Prefix.String(),
						nhs,
						tunnels,
						labels,
					)

					dumpedRouteRecords = append(dumpedRouteRecords, dumpedRouteRecord)
				}

			case *memclnt.ControlPingReply:
				break LoopRoute

			default:
				return dumpedRouteRecords, fmt.Errorf("unexpected message type %T", msg)
			}
		}
	}

	return dumpedRouteRecords, nil
}

// LookupFIPRoute lookups specific IP/MPLS route to floating IP address and fill TunnelID/Label in the VPPIPRoute structure
func LookupFIPRoute(stream api.Stream, vppIPRoute *model.VPPIPRoute) (bool, error) {
	isFound := false

	floatingIPPrefix, err := ip_types.ParsePrefix(vppIPRoute.Prefix)
	if err != nil {
		return false, err
	}

	req := &ip.IPRouteLookupV2{
		TableID: vppIPRoute.VRFID,
		Exact:   0,
		Prefix:  floatingIPPrefix,
	}

	if err = stream.SendMsg(req); err != nil {
		return false, err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return false, err
	}

	reply := msg.(*ip.IPRouteLookupV2Reply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return false, api.RetvalToVPPApiError(reply.Retval)
	}

	tunnels := make([]uint32, 0)

	labels := make([]uint32, 0)

	for _, p := range reply.Route.Paths {
		tunnel := p.Nh.ObjID / 16777216 // div 2^24

		tunnels = append(tunnels, tunnel)

		label := p.LabelStack[0].Label

		labels = append(labels, label)
	}

	if reply.Route.Paths[0].Type.String() == "FIB_API_PATH_TYPE_UDP_ENCAP" { //nolint:goconst
		isFound = true
		vppIPRoute.TunnelIDs = tunnels
		vppIPRoute.FIPMPLSLabels = labels
	}

	return isFound, nil
}

// AddDelFIPRoute adds/deletes IP/MPLS route to floating IP of VM via vRouter.
// (ip route add|del <fip>/32 via <vrouter> udp-encap <id> mpls-lookup-in-table 0 out-labels <mpls_label>) and
// (ip route add|del <fip>/32 via <vrouter> <vpp_main_interface_name> udp-encap <id> <vpp_main_interface_name> out-labels <mpls_label>)
func AddDelFIPRoute(stream api.Stream, isAdd bool, vppIPRoute *model.VPPIPRoute) error {
	for _, p := range vppIPRoute.TunnelIDs {
		if p == model.UndefinedTunnelID {
			return fmt.Errorf("wrong udp tunnel id %d", p)
		}
	}

	floatingPrefix, err := ip_types.ParsePrefix(vppIPRoute.Prefix)
	if err != nil {
		return err
	}

	nextHops := make([]ip_types.IP4Address, len(vppIPRoute.NextHops))

	for i, nh := range vppIPRoute.NextHops {
		nhIP, err := ip_types.ParseIP4Address(nh)
		if err != nil {
			return err
		}

		nextHops[i] = nhIP
	}

	paths := make([]fib_types.FibPath, len(vppIPRoute.NextHops))

	for i := range vppIPRoute.NextHops {
		paths[i].TableID = 0                                    // mpls tunnel belongs to global routing table
		paths[i].SwIfIndex = uint32(vppIPRoute.MainInterfaceID) // vpp main interface
		paths[i].Type = fib_types.FIB_API_PATH_TYPE_UDP_ENCAP
		paths[i].Flags = fib_types.FIB_API_PATH_FLAG_NONE
		paths[i].Proto = fib_types.FIB_API_PATH_NH_PROTO_IP4
		paths[i].Nh = fib_types.FibPathNh{
			Address: ip_types.AddressUnionIP4(nextHops[i]),
			ObjID:   vppIPRoute.TunnelIDs[i],
		}
		paths[i].NLabels = 1
		paths[i].LabelStack = [16]fib_types.FibMplsLabel{
			{
				IsUniform: 0,
				TTL:       64,
				Exp:       0,
				Label:     vppIPRoute.FIPMPLSLabels[i],
			},
		}
	}

	isMultipath := false

	if len(vppIPRoute.NextHops) > 1 {
		isMultipath = true
	}

	req := &ip.IPRouteAddDelV2{
		IsAdd:       isAdd,
		IsMultipath: isMultipath,
		Route: ip.IPRouteV2{
			TableID: vppIPRoute.VRFID,
			Prefix:  floatingPrefix,
			NPaths:  uint8(len(vppIPRoute.NextHops)),
			Paths:   paths,
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*ip.IPRouteAddDelV2Reply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// AddDelMPLSLocalLabelRoute adds/deletes MPLS local-label route to accept labeled traffic from vRouters and send it to physical network
// (0.0.0.0/0 with local-label assigned via physical network).
func AddDelMPLSLocalLabelRoute(stream api.Stream, isAdd bool, vppVRFTable model.VPPVRFTable) error {
	phyNetNhAddr, err := ip_types.ParseIP4Address(vppVRFTable.NextHop)
	if err != nil {
		return err
	}

	req := &mpls.MplsRouteAddDel{
		MrIsAdd:       isAdd,
		MrIsMultipath: false,
		MrRoute: mpls.MplsRoute{
			MrTableID:     0, // MPLS table always belongs to global table
			MrLabel:       vppVRFTable.MPLSLocalLabel,
			MrEos:         1,
			MrEosProto:    uint8(fib_types.FIB_API_PATH_NH_PROTO_IP4),
			MrIsMulticast: false,
			MrNPaths:      1,
			MrPaths: []fib_types.FibPath{
				{
					SwIfIndex: uint32(vppVRFTable.SubInterfaceID), // Sub-interface of specific VRF
					Proto:     fib_types.FIB_API_PATH_NH_PROTO_IP4,
					Type:      fib_types.FIB_API_PATH_TYPE_NORMAL,
					Flags:     fib_types.FIB_API_PATH_FLAG_NONE,
					Nh: fib_types.FibPathNh{
						Address: ip_types.AddressUnionIP4(phyNetNhAddr),
					},
					NLabels:    0,
					LabelStack: [16]fib_types.FibMplsLabel{},
				},
			},
		},
	}

	if err = stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*mpls.MplsRouteAddDelReply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// DumpMPLSLocalLabelRoute dumps all MPLS local-label routes from VPP
func DumpMPLSLocalLabelRoute(stream api.Stream) ([]mpls.MplsRouteDetails, error) {
	var dumpedMplsRoutes []mpls.MplsRouteDetails

	req := &mpls.MplsRouteDump{
		Table: mpls.MplsTable{
			MtTableID: 0,
		},
	}

	if err := stream.SendMsg(req); err != nil {
		return nil, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return nil, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return nil, err
		}

		switch replay := msg.(type) {
		case *mpls.MplsRouteDetails:
			dumpedMplsRoutes = append(dumpedMplsRoutes, *replay)

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return nil, fmt.Errorf("unexpected message type received: %T", msg)
		}
	}

	return dumpedMplsRoutes, nil
}

// GetRoutingTables gets all ipv4 routing tables
func GetRoutingTables(stream api.Stream) ([]ip.IPTableDetails, error) {
	var dumpedTables []ip.IPTableDetails

	req := &ip.IPTableDump{}

	if err := stream.SendMsg(req); err != nil {
		return nil, err
	}

	if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return nil, err
	}

LoopTable:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return nil, err
		}

		switch replay := msg.(type) {
		case *ip.IPTableDetails:
			if !replay.Table.IsIP6 {
				dumpedTables = append(dumpedTables, *replay)
			}

		case *memclnt.ControlPingReply:
			break LoopTable

		default:
			return nil, fmt.Errorf("unexpected message type received: %T", msg)
		}
	}

	return dumpedTables, nil
}

// DumpIPRoutes gets and parse all IPv4 routes for all VRF from VPP to External network (exclude route to FIP).
func DumpIPRoutes(stream api.Stream) ([]model.VPPIPRoute, error) {
	var (
		dumpedRoutes []model.VPPIPRoute
		dumpedRoute  model.VPPIPRoute
		dumpedTables []ip.IPTableDetails
	)

	// get all ipv4 routing tables (not routes!)

	{
		req := &ip.IPTableDump{}

		if err := stream.SendMsg(req); err != nil {
			return nil, err
		}

		if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
			return nil, err
		}

	LoopTable:
		for {
			msg, err := stream.RecvMsg()
			if err != nil {
				return nil, err
			}

			switch replay := msg.(type) {
			case *ip.IPTableDetails:
				if !replay.Table.IsIP6 {
					dumpedTables = append(dumpedTables, *replay)
				}

			case *memclnt.ControlPingReply:
				break LoopTable

			default:
				return nil, fmt.Errorf("unexpected message type received: %T", msg)
			}
		}
	}

	// get all ipv4 routes for each table

	{
		for _, table := range dumpedTables {
			req := &ip.IPRouteV2Dump{
				Table: table.Table,
			}

			if err := stream.SendMsg(req); err != nil {
				return nil, err
			}

			if err := stream.SendMsg(&memclnt.ControlPing{}); err != nil {
				return nil, err
			}

		LoopRoute:
			for {
				msg, err := stream.RecvMsg()
				if err != nil {
					return nil, err
				}

				switch replay := msg.(type) {
				case *ip.IPRouteV2Details:
					// Get only external IPv4 route by type and specific attributes:
					if replay.Route.Paths[0].Type.String() == "FIB_API_PATH_TYPE_NORMAL" && //nolint:goconst
						// exclude connected interface
						replay.Route.Paths[0].Nh.Address.GetIP4().String() != "0.0.0.0" &&
						// exclude local interface address
						netutils.Addr(replay.Route.Prefix.String()) != replay.Route.Paths[0].Nh.Address.GetIP4().String() &&
						// exclude all /32 for vRouter's UDP tunnels
						netutils.MaskLen(replay.Route.Prefix.String()) != 32 {

						dumpedRoute.VRFID = replay.Route.TableID
						dumpedRoute.Prefix = replay.Route.Prefix.String()

						if dumpedRoute.VRFID == 0 {
							dumpedRoute.MainInterfaceID = interface_types.InterfaceIndex(replay.Route.Paths[0].SwIfIndex)
						} else {
							dumpedRoute.SubInterfaceID = interface_types.InterfaceIndex(replay.Route.Paths[0].SwIfIndex)
						}

						for _, nhIP := range replay.Route.Paths {
							dumpedRoute.NextHops = append(dumpedRoute.NextHops, nhIP.Nh.Address.GetIP4().String())
						}

						dumpedRoutes = append(dumpedRoutes, dumpedRoute)
					}

				case *memclnt.ControlPingReply:
					break LoopRoute

				default:
					return nil, fmt.Errorf("unexpected message type received: %T", msg)
				}
			}
		}
	}

	return dumpedRoutes, nil
}

// CountRoutesPerTable counts the IPv4 and floating IP route records for specific VRF (Table) ID (used for Metric Exporter)
func CountRoutesPerTable(stream api.Stream, tableID ip.IPTable) (
	ipRouteCount, fipRouteCount float64, err error,
) {
	req := &ip.IPRouteV2Dump{
		Table: tableID,
	}

	if err = stream.SendMsg(req); err != nil {
		return 0, 0, err
	}

	if err = stream.SendMsg(&memclnt.ControlPing{}); err != nil {
		return 0, 0, err
	}

Loop:
	for {
		msg, err := stream.RecvMsg()
		if err != nil {
			return 0, 0, err
		}

		switch replay := msg.(type) {
		case *ip.IPRouteV2Details:
			if replay.Route.Paths[0].Type.String() == "FIB_API_PATH_TYPE_NORMAL" {
				ipRouteCount++

				continue
			}

			if replay.Route.Paths[0].Type.String() == "FIB_API_PATH_TYPE_UDP_ENCAP" {
				fipRouteCount++

				continue
			}

		case *memclnt.ControlPingReply:
			break Loop

		default:
			return 0, 0, fmt.Errorf("unexpected message type received: %T", msg)
		}
	}

	return ipRouteCount, fipRouteCount, nil
}

// AddDelIPRoute adds/deletes an IPv4 route from VPP (route to Internet/External Networks or to vRouters)
func AddDelIPRoute(stream api.Stream, isAdd bool, vppIPRoute model.VPPIPRoute) error {
	prefix, err := ip_types.ParsePrefix(vppIPRoute.Prefix)
	if err != nil {
		return err
	}

	var swInterfaceIndex uint32

	nextHops := make([]ip_types.IP4Address, len(vppIPRoute.NextHops))

	for i, nh := range vppIPRoute.NextHops {
		ipNH, err := ip_types.ParseIP4Address(nh)
		if err != nil {
			return err
		}

		nextHops[i] = ipNH
	}

	if vppIPRoute.VRFID == 0 {
		swInterfaceIndex = uint32(vppIPRoute.MainInterfaceID)
	} else {
		if vppIPRoute.SubInterfaceID == model.UndefinedSubIf {
			return fmt.Errorf("sub-interface %d not defined", vppIPRoute.SubInterfaceID)
		}

		swInterfaceIndex = uint32(vppIPRoute.SubInterfaceID)
	}

	paths := make([]fib_types.FibPath, len(vppIPRoute.NextHops))

	for i := range vppIPRoute.NextHops {
		paths[i].TableID = vppIPRoute.VRFID
		paths[i].SwIfIndex = swInterfaceIndex
		paths[i].Type = fib_types.FIB_API_PATH_TYPE_NORMAL
		paths[i].Flags = fib_types.FIB_API_PATH_FLAG_NONE
		paths[i].Proto = fib_types.FIB_API_PATH_NH_PROTO_IP4
		paths[i].Nh = fib_types.FibPathNh{Address: ip_types.AddressUnionIP4(nextHops[i])}
	}

	req := &ip.IPRouteAddDelV2{
		IsAdd:       isAdd,
		IsMultipath: true,
		Route: ip.IPRouteV2{
			TableID: vppIPRoute.VRFID,
			Prefix:  prefix,
			NPaths:  uint8(len(vppIPRoute.NextHops)),
			Paths:   paths,
		},
	}

	if err = stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*ip.IPRouteAddDelV2Reply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

// AddBlackHoleIPRoute creates blackHole IPv4 route for Aggregated floating IP prefix to avoid loops
func AddBlackHoleIPRoute(stream api.Stream, vppIPRoute model.VPPIPRoute) error {
	fipPrefix, err := ip_types.ParsePrefix(vppIPRoute.Prefix)
	if err != nil {
		return fmt.Errorf("failed to parse prefix %s: %w", vppIPRoute.Prefix, err)
	}

	req := &ip.IPRouteAddDelV2{
		IsAdd: true,
		Route: ip.IPRouteV2{
			TableID: vppIPRoute.VRFID,
			Prefix:  fipPrefix,
			NPaths:  1,
			Src:     9, // imitates CLI, but really it is API :-)
			Paths: []fib_types.FibPath{
				{
					SwIfIndex: math.MaxUint32, // null interface
					TableID:   0,
					Type:      fib_types.FIB_API_PATH_TYPE_NORMAL,
					Flags:     fib_types.FIB_API_PATH_FLAG_NONE,
					Proto:     fib_types.FIB_API_PATH_NH_PROTO_IP4,
				},
			},
		},
	}

	if err = stream.SendMsg(req); err != nil {
		return err
	}

	msg, err := stream.RecvMsg()
	if err != nil {
		return err
	}

	reply := msg.(*ip.IPRouteAddDelV2Reply)

	if api.RetvalToVPPApiError(reply.Retval) != nil {
		return api.RetvalToVPPApiError(reply.Retval)
	}

	return nil
}

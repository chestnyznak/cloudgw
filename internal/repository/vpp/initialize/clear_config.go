package initialize

import (
	"fmt"

	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"go.fd.io/govpp/api"

	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
)

// ClearVPPConfig clears and deletes all vpp settings (interfaces, routes, tables, vrfs)
func ClearVPPConfig(stream api.Stream, mainInterfaceID uint32, vppVRFStorage *imdb.VPPVRFStorage) error {
	// delete existing non-host ipv4 routes for all vrf (needed to correctly delete mpls local route)
	dumpedIPRoutes, err := vpp.DumpIPRoutes(stream)
	if err != nil {
		return fmt.Errorf("failed to dump ip routes from vpp: %w", err)
	}

	for _, route := range dumpedIPRoutes {
		if err = vpp.AddDelIPRoute(stream, false, route); err != nil {
			return fmt.Errorf("failed to delete ip route %s from vpp: %w", route.Prefix, err)
		}
	}

	// delete mpls local-label routes for non-default vrf if they exists

	vppVRFs := vppVRFStorage.GetVRFs()

	for _, table := range vppVRFs {
		if table.ID == 0 {
			continue
		}

		dumpedMplsRoute, err := vpp.DumpMPLSLocalLabelRoute(stream)
		if err != nil {
			return fmt.Errorf("failed to dump mpls local route label id %d from vpp: %w", table.MPLSLocalLabel, err)
		}

		if len(dumpedMplsRoute) == 0 {
			continue
		}

		if err = vpp.AddDelMPLSLocalLabelRoute(stream, false, *table); err != nil {
			return fmt.Errorf("failed to delete mpls local route label id %d from vpp: %w", table.MPLSLocalLabel, err)
		}
	}

	// delete all sub-interfaces

	if _, err = vpp.DelSubInterfaces(stream, mainInterfaceID); err != nil {
		return fmt.Errorf("failed to delete sub-interfaces: %w", err)
	}

	// get all existing udp tunnel ids

	dumpedUDPTunnels, err := vpp.DumpUDPTunnels(stream)
	if err != nil {
		return fmt.Errorf("failed to dump udp tunnels from vpp: %w", err)
	}

	// get all existing ip/mpls routes to vm floating ip addresses

	dumpedFIPRoutes, err := vpp.DumpFIPRoutes(stream)
	if err != nil {
		return fmt.Errorf("failed to dump vm floating ip routes from vpp: %w", err)
	}

	// unconditionally delete all existing matching routes to fips and udp tunnels

	for _, routeRecord := range dumpedFIPRoutes {
		for _, udpRecord := range dumpedUDPTunnels {
			if contains(routeRecord.TunnelIDs, udpRecord.TunnelID) {
				if err = vpp.AddDelFIPRoute(stream, false, &routeRecord); err != nil {
					return fmt.Errorf("failed to remove mpls/floating ip route %s: %w", routeRecord.Prefix, err)
				}

				if err = vpp.DelUDPTunnel(stream, udpRecord.TunnelID); err != nil {
					return fmt.Errorf("failed to delete udp tunnel id %d: %w", udpRecord.TunnelID, err)
				}
			}
		}
	}

	// remove all "orphan" udp tunnel id records (records with no routes were found)

	dumpedUDPTunnels, err = vpp.DumpUDPTunnels(stream)
	if err != nil {
		return fmt.Errorf("failed to dump udp tunnels from vpp: %w", err)
	}

	for _, udpRecord := range dumpedUDPTunnels {
		if err = vpp.DelUDPTunnel(stream, udpRecord.TunnelID); err != nil {
			return fmt.Errorf("failed to delete udp tunnel id %d from vpp: %w", udpRecord.TunnelID, err)
		}
	}

	// delete all "orphan" floating ip routes

	dumpedFIPRoutes, err = vpp.DumpFIPRoutes(stream)
	if err != nil {
		return fmt.Errorf("failed to dump vm floating ip routes: %w", err)
	}

	for _, routeRecord := range dumpedFIPRoutes {
		if err = vpp.AddDelFIPRoute(stream, false, &routeRecord); err != nil {
			return fmt.Errorf("failed to remove mpls/floating ip route %s from vpp: %w", routeRecord.Prefix, err)
		}
	}

	// reset vpp main interface configuration (delete ip address, disable)

	if err = vpp.ResetMainInterface(stream, mainInterfaceID); err != nil {
		return fmt.Errorf("failed to reset main interface: %w", err)
	}

	// delete all vpp vrfs except default vrf

	for _, table := range vppVRFs {
		if table.ID == 0 {
			continue
		}

		if err = vpp.AddDelVRF(stream, false, *table); err != nil {
			return fmt.Errorf("failed to delete vpp routing table id %d: %w", table.ID, err)
		}
	}

	return nil
}

func contains(elements []uint32, target uint32) bool {
	for _, e := range elements {
		if e == target {
			return true
		}
	}

	return false
}

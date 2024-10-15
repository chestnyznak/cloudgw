package initialize

import (
	"fmt"

	"go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/interface_types"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
)

// AddVPPInitConfig creates initial vpp static config (vrf, tables, interfaces, static routes)
func AddVPPInitConfig(stream api.Stream, vppVRFStorage *imdb.VPPVRFStorage, vppMainInterfaceID uint32, vppMainInterfaceGW string) error {
	vppVRFs := vppVRFStorage.GetVRFs()

	if len(vppVRFs) < 2 {
		return fmt.Errorf("found %d vrf(s) in storage (need at least 2), check yaml config file", len(vppVRFs))
	}

	// create vpp vrfs

	for _, table := range vppVRFs {
		if table.ID == 0 {
			continue
		}

		if err := vpp.AddDelVRF(stream, true, *table); err != nil {
			return fmt.Errorf("failed to create vpp vrf id %d in vpp: %w", table.ID, err)
		}
	}

	// create vpp mpls table

	if err := vpp.AddDelMPLSTable(stream, true); err != nil {
		return fmt.Errorf("failed to create mpls fib in vrf: %w", err)
	}

	// configure vpp main interface configuration (ip address, enable, mpls support)

	if err := vpp.SetupMainInterface(stream, *vppVRFs[0]); err != nil {
		return fmt.Errorf("failed to setup main interface in vpp: %w", err)
	}

	// create sub-interface to physical network for all vrf (except grt)

	for _, table := range vppVRFs {
		if table.ID == 0 {
			continue
		}

		createdSubIf, err := vpp.AddSubInterface(stream, table)
		if err != nil {
			return fmt.Errorf("failed to create sub-interface %d: %w", table.SubInterfaceID, err)
		}

		table.SubInterfaceID = createdSubIf
	}

	// register mpls over udp decapsulation port 6635

	if err := vpp.AddUDPDecap(stream); err != nil {
		return fmt.Errorf("failed to setup udp decap: %w", err)
	}

	// create mpls local-label route to accept labeled traffic from vRouters

	for _, table := range vppVRFs {
		if table.ID == 0 {
			continue
		}

		if err := vpp.AddDelMPLSLocalLabelRoute(stream, true, *table); err != nil {
			return fmt.Errorf("failed to add mpls local label in table %d: %w", table.ID, err)
		}
	}

	// create default ipv4 route to vrouters from vpp grt

	if err := vpp.AddDelIPRoute(
		stream,
		true,
		model.VPPIPRoute{
			VRFID:           0,
			Prefix:          "0.0.0.0/0",
			NextHops:        []string{vppMainInterfaceGW},
			MainInterfaceID: interface_types.InterfaceIndex(vppMainInterfaceID),
			SubInterfaceID:  model.UndefinedSubIf,
			TunnelIDs:       []uint32{model.UndefinedTunnelID},
			FIPMPLSLabels:   []uint32{model.UndefinedLabel},
		},
	); err != nil {
		return fmt.Errorf("failed to add grt default route: %w", err)
	}

	// create floating ip aggregated routes (used for black-hole routes)

	var vppAggrFIPRoutes []model.VPPIPRoute

	for _, vrf := range vppVRFs {
		for _, prefix := range vrf.FIPPrefixes {
			vppAggrFIPRoute := model.NewVPPIPRoute(
				vrf.ID,
				interface_types.InterfaceIndex(vppMainInterfaceID),
				model.UndefinedSubIf,
				prefix,
				[]string{""},
				[]uint32{model.UndefinedTunnelID},
				[]uint32{model.UndefinedLabel},
			)

			vppAggrFIPRoutes = append(vppAggrFIPRoutes, vppAggrFIPRoute)
		}
	}

	// add black-hole routes

	for _, route := range vppAggrFIPRoutes {
		if err := vpp.AddBlackHoleIPRoute(stream, route); err != nil {
			return fmt.Errorf("failed to create blackhole floating ip route in vrf %d: %w", route.VRFID, err)
		}
	}

	return nil
}

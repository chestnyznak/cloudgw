package gobgp

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"fmt"
	"strings"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/types/known/anypb"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
	"git.crptech.ru/cloud/cloudgw/pkg/logger/gobgplogger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

// CreateGoBGPServer starts local GoBGP server with specific level logging
func CreateGoBGPServer(listenAddress string, logLevel, logFormat, logOutput string) *server.BgpServer {
	bgpLogger := gobgplogger.NewGoBGPLogger(logLevel, logFormat, logOutput)

	s := server.NewBgpServer(server.GrpcListenAddress(listenAddress), server.LoggerOption(bgpLogger))

	go s.Serve()

	return s
}

// SetGoBGPLocalConfig creates GoBGP local configuration on local GoBGP server
func SetGoBGPLocalConfig(ctx context.Context, srv *server.BgpServer, asn uint32, routerID string, bgpLocalPort int32) error {
	req := &bgpapi.StartBgpRequest{
		Global: &bgpapi.Global{
			Asn:        asn,
			RouterId:   routerID,
			ListenPort: bgpLocalPort,
		},
	}
	if err := srv.StartBgp(ctx, req); err != nil {
		return err
	}

	return nil
}

// GetGoBGPLocalConfig gets local GoBGP configuration of local GoBGP server
func GetGoBGPLocalConfig(ctx context.Context, srv *server.BgpServer) (*bgpapi.GetBgpResponse, error) {
	resp, err := srv.GetBgp(ctx, &bgpapi.GetBgpRequest{})
	if err != nil {
		return nil, err
	}

	return resp, err
}

// AddGoBGPVRF creates GoBGP VRF on local GoBGP server
func AddGoBGPVRF(ctx context.Context, srv *server.BgpServer, vrf *model.BGPVRFTable) error {
	vrfTable := &bgpapi.Vrf{
		Name:     vrf.Name,
		Id:       vrf.ID,
		Rd:       vrf.RD,
		ImportRt: vrf.ImportRT,
		ExportRt: vrf.ExportRT,
	}

	if err := srv.AddVrf(ctx, &bgpapi.AddVrfRequest{
		Vrf: vrfTable,
	}); err != nil {
		return err
	}

	return nil
}

// DelGoBGPVRF deletes GoBGP VRF on local GoBGP server
func DelGoBGPVRF(ctx context.Context, srv *server.BgpServer, vrfName string) error {
	if err := srv.DeleteVrf(ctx, &bgpapi.DeleteVrfRequest{
		Name: vrfName,
	}); err != nil {
		return err
	}

	return nil
}

// AddBGPPeer creates BGP Peer on local GoBGP server
func AddBGPPeer(ctx context.Context, srv *server.BgpServer, peer *model.BGPPeer) error {
	neigh := &bgpapi.Peer{
		Conf: &bgpapi.PeerConf{
			NeighborAddress: peer.PeerAddress,
			PeerAsn:         peer.PeerASN,
			AuthPassword:    peer.Md5Password,
			Vrf:             peer.VRFName,
		},
		EbgpMultihop: &bgpapi.EbgpMultihop{
			Enabled:     peer.EbgpMultiHop,
			MultihopTtl: peer.EbgpMultiHopTTL,
		},
		Timers: &bgpapi.Timers{
			Config: &bgpapi.TimersConfig{
				KeepaliveInterval: peer.KeepAliveTimer,
				HoldTime:          peer.HoldTimer,
			},
		},
		Transport: &bgpapi.Transport{
			RemotePort: peer.PeerPort,
		},
		AfiSafis: []*bgpapi.AfiSafi{
			{
				Config: &bgpapi.AfiSafiConfig{
					Family: &bgpapi.Family{
						Afi:  peer.AFI,
						Safi: peer.SAFI,
					},
				},
			},
		},
	}

	if err := srv.AddPeer(ctx, &bgpapi.AddPeerRequest{
		Peer: neigh,
	}); err != nil {
		return err
	}

	return nil
}

// DelBGPPeer deletes a BGP Peer on local GoBGP server
func DelBGPPeer(ctx context.Context, srv *server.BgpServer, peer *model.BGPPeer) error {
	if err := srv.DeletePeer(ctx, &bgpapi.DeletePeerRequest{
		Address: peer.PeerAddress,
	}); err != nil {
		return err
	}

	return nil
}

// AdvWdrawIPv4Prefix advertises/withdraws IPv4/Unicast prefix on local GoBGP server in GRT
func AdvWdrawIPv4Prefix(ctx context.Context, srv *server.BgpServer, isAdvertise bool, ipv4BGPNLRIAttrs gobgpapi.BGPNLRIAttrs) error {
	nlri, _ := anypb.New(&bgpapi.IPAddressPrefix{
		Prefix:    netutils.Addr(ipv4BGPNLRIAttrs.Prefix),
		PrefixLen: netutils.MaskLen(ipv4BGPNLRIAttrs.Prefix),
	})

	origin, _ := anypb.New(&bgpapi.OriginAttribute{
		Origin: 2,
	})

	nh, _ := anypb.New(&bgpapi.NextHopAttribute{
		NextHop: ipv4BGPNLRIAttrs.NextHop,
	})

	pattrs := []*anypb.Any{origin, nh}

	if isAdvertise {
		// Advertise the prefix
		if _, err := srv.AddPath(ctx, &bgpapi.AddPathRequest{
			TableType: bgpapi.TableType_GLOBAL,
			Path: &bgpapi.Path{
				Nlri:   nlri,
				Pattrs: pattrs,
				Family: &bgpapi.Family{
					Afi:  bgpapi.Family_AFI_IP,
					Safi: bgpapi.Family_SAFI_UNICAST,
				},
			},
		}); err != nil {
			return err
		}
	} else {
		// Withdraw the prefix
		if err := srv.DeletePath(ctx, &bgpapi.DeletePathRequest{
			TableType: bgpapi.TableType_GLOBAL,
			Path: &bgpapi.Path{
				Nlri:   nlri,
				Pattrs: pattrs,
				Family: &bgpapi.Family{
					Afi:  bgpapi.Family_AFI_IP,
					Safi: bgpapi.Family_SAFI_UNICAST,
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

// AdvWdrawVpnv4Prefix advertises/withdraws VPNv4 prefix on local GoBGP server
func AdvWdrawVpnv4Prefix(ctx context.Context, srv *server.BgpServer, isAdvertise bool, bgpNLRIAttrs gobgpapi.BGPNLRIAttrs, sourceASN uint32) error {
	nlri, _ := anypb.New(&bgpapi.LabeledVPNIPAddressPrefix{
		Labels:    bgpNLRIAttrs.MPLSLabel,
		Rd:        bgpNLRIAttrs.RD,
		Prefix:    netutils.Addr(bgpNLRIAttrs.Prefix),
		PrefixLen: netutils.MaskLen(bgpNLRIAttrs.Prefix),
	})

	origin, _ := anypb.New(&bgpapi.OriginAttribute{
		Origin: 2,
	})

	med, _ := anypb.New(&bgpapi.MultiExitDiscAttribute{
		Med: 0,
	})

	localPref, _ := anypb.New(&bgpapi.LocalPrefAttribute{
		LocalPref: 100,
	})

	rt := bgpNLRIAttrs.RT

	tunnelType, _ := anypb.New(&bgpapi.EncapExtended{
		TunnelType: 13, // MPLS over UDP
	})

	communities, _ := anypb.New(&bgpapi.ExtendedCommunitiesAttribute{
		Communities: []*anypb.Any{rt[0], tunnelType},
	})

	nlris := []*anypb.Any{nlri}

	nlriAttr, _ := anypb.New(&bgpapi.MpReachNLRIAttribute{
		Family: &bgpapi.Family{
			Afi:  bgpapi.Family_AFI_IP,
			Safi: bgpapi.Family_SAFI_MPLS_VPN,
		},
		Nlris:    nlris,
		NextHops: []string{bgpNLRIAttrs.NextHop},
	})

	// Add ASN of source of the route to avoid BGP loop
	asnPath, _ := anypb.New(&bgpapi.AsPathAttribute{
		Segments: []*bgpapi.AsSegment{
			{
				Type:    2,
				Numbers: []uint32{sourceASN},
			},
		},
	})

	pAttrs := []*anypb.Any{origin, med, localPref, communities, nlriAttr, asnPath}

	if isAdvertise {
		// Advertise the prefix
		if _, err := srv.AddPath(ctx, &bgpapi.AddPathRequest{
			TableType: bgpapi.TableType_GLOBAL,
			Path: &bgpapi.Path{
				Nlri:   nlri,
				Pattrs: pAttrs,
				Family: &bgpapi.Family{
					Afi:  bgpapi.Family_AFI_IP,
					Safi: bgpapi.Family_SAFI_MPLS_VPN,
				},
			},
		}); err != nil {
			return err
		}
	} else {
		// Withdraw the prefix
		if err := srv.DeletePath(ctx, &bgpapi.DeletePathRequest{
			TableType: bgpapi.TableType_GLOBAL,
			Path: &bgpapi.Path{
				Nlri:   nlri,
				Pattrs: pAttrs,
				Family: &bgpapi.Family{
					Afi:  bgpapi.Family_AFI_IP,
					Safi: bgpapi.Family_SAFI_MPLS_VPN,
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

// CreateGoBGPPrefixSet creates GoBGP PrefixSets (aka Prefix-List)
func CreateGoBGPPrefixSet(ctx context.Context, srv *server.BgpServer, prefix string, minMaskLen uint32, maxMaskLen uint32) (*bgpapi.DefinedSet, error) {
	pref := bgpapi.Prefix{
		IpPrefix:      prefix,
		MaskLengthMin: minMaskLen,
		MaskLengthMax: maxMaskLen,
	}

	dsName := strings.Join([]string{prefix, fmt.Sprint(minMaskLen), fmt.Sprint(maxMaskLen)}, "-")

	ds := &bgpapi.DefinedSet{
		DefinedType: bgpapi.DefinedType_PREFIX,
		Name:        dsName,
		Prefixes:    []*bgpapi.Prefix{&pref},
	}

	if err := srv.AddDefinedSet(ctx, &bgpapi.AddDefinedSetRequest{
		DefinedSet: ds,
	}); err != nil {
		return nil, err
	}

	return ds, nil
}

// CreateGoBGPNeighborSet creates GoBGP NeighborSets  (neighbor format like "203.0.113.1/32")
func CreateGoBGPNeighborSet(ctx context.Context, srv *server.BgpServer, neighbors []string) (*bgpapi.DefinedSet, error) {
	nsName := strings.Join(neighbors, "-")

	ds := &bgpapi.DefinedSet{
		DefinedType: bgpapi.DefinedType_NEIGHBOR,
		Name:        nsName,
		List:        neighbors,
	}

	if err := srv.AddDefinedSet(ctx, &bgpapi.AddDefinedSetRequest{
		DefinedSet: ds,
	}); err != nil {
		return nil, err
	}

	return ds, nil
}

// CreateGoBGPNPolicyStatements create GoBGP statement for a PrefixSet and a NeighborSes with specific action
func CreateGoBGPNPolicyStatements(prefixSet *bgpapi.DefinedSet, neighborSet *bgpapi.DefinedSet, action bgpapi.RouteAction) (*bgpapi.Statement, error) {
	stNameMd5 := md5.Sum([]byte(strings.Join([]string{prefixSet.Name, neighborSet.Name}, "-"))) //nolint:gosec

	st := &bgpapi.Statement{
		Name: hex.EncodeToString(stNameMd5[:]),

		Conditions: &bgpapi.Conditions{
			PrefixSet: &bgpapi.MatchSet{
				Name: prefixSet.Name,
			},

			NeighborSet: &bgpapi.MatchSet{
				Name: neighborSet.Name,
			},
		},

		Actions: &bgpapi.Actions{
			RouteAction: action,
		},
	}

	return st, nil
}

// AddBGPGlobalExportPolicy creates GoBGP global export policy
func AddBGPGlobalExportPolicy(ctx context.Context, srv *server.BgpServer, statements []*bgpapi.Statement, defaultAction bgpapi.RouteAction) error {
	var str strings.Builder

	for _, st := range statements {
		str.WriteString(st.Name)
	}

	polNameMd5 := md5.Sum([]byte(str.String())) //nolint:gosec

	expPol := &bgpapi.Policy{
		Name:       hex.EncodeToString(polNameMd5[:]),
		Statements: statements,
	}

	if err := srv.AddPolicy(ctx, &bgpapi.AddPolicyRequest{Policy: expPol}); err != nil {
		return err
	}

	if err := srv.AddPolicyAssignment(ctx, &bgpapi.AddPolicyAssignmentRequest{
		Assignment: &bgpapi.PolicyAssignment{
			Name:          "global", // as VPNv4 works in GRT
			Direction:     bgpapi.PolicyDirection_EXPORT,
			Policies:      []*bgpapi.Policy{expPol},
			DefaultAction: defaultAction,
		},
	}); err != nil {
		return err
	}

	return nil
}

// CreateGoBGPPolicy create typical cloudgw bgp policy (deny /32 and 0/0 to physical network, deny aggregated floating ip to tungsten fabric, allow any other)
func CreateGoBGPPolicy(ctx context.Context, srv *server.BgpServer, allTFControllerIPs, allPhyNetRouterIPs []string) error {
	// create prefix sets (aka prefix-list)
	allHostRoutes, err := CreateGoBGPPrefixSet(ctx, srv, "0.0.0.0/0", 32, 32)
	if err != nil {
		return fmt.Errorf("failed to create prefix set: %w", err)
	}

	defaultRoute, err := CreateGoBGPPrefixSet(ctx, srv, "0.0.0.0/0", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to create prefix set: %w", err)
	}

	allExceptDefaultRoute, err := CreateGoBGPPrefixSet(ctx, srv, "0.0.0.0/0", 1, 32)
	if err != nil {
		return fmt.Errorf("failed to create prefix set: %w", err)
	}

	// create neighbor sets

	allTFControllers, err := CreateGoBGPNeighborSet(ctx, srv, allTFControllerIPs)
	if err != nil {
		return fmt.Errorf("failed to create neighbor set: %w", err)
	}

	allPhyNetRouters, err := CreateGoBGPNeighborSet(ctx, srv, allPhyNetRouterIPs)
	if err != nil {
		return fmt.Errorf("failed to create neighbor set: %w", err)
	}

	// create policy statements

	st1, err := CreateGoBGPNPolicyStatements(allHostRoutes, allPhyNetRouters, bgpapi.RouteAction_REJECT)
	if err != nil {
		return fmt.Errorf("failed to create policy statement: %w", err)
	}

	st2, err := CreateGoBGPNPolicyStatements(
		defaultRoute,
		allPhyNetRouters,
		bgpapi.RouteAction_REJECT,
	)
	if err != nil {
		return fmt.Errorf("failed to create policy statement: %w", err)
	}

	st3, err := CreateGoBGPNPolicyStatements(allExceptDefaultRoute, allTFControllers, bgpapi.RouteAction_REJECT)
	if err != nil {
		return fmt.Errorf("failed to create policy statement: %w", err)
	}

	// apply as export policy

	if err = AddBGPGlobalExportPolicy(
		ctx,
		srv,
		[]*bgpapi.Statement{st1, st2, st3},
		bgpapi.RouteAction_ACCEPT,
	); err != nil {
		return fmt.Errorf("failed to add global export policy: %w", err)
	}

	return nil
}

// UpdateGoBGPPerPeerMetrics returns number of received/sent updates for specific peer
func UpdateGoBGPPerPeerMetrics(ctx context.Context, srv *server.BgpServer, tableType bgpapi.TableType, afi bgpapi.Family_Afi, safi bgpapi.Family_Safi, bgpPeer string) (float64, error) {
	req := &bgpapi.GetTableRequest{
		TableType: tableType,
		Family: &bgpapi.Family{
			Afi:  afi,
			Safi: safi,
		},
		Name: bgpPeer,
	}

	result, err := srv.GetTable(ctx, req)
	if err != nil {
		return 0, err
	}

	return float64(result.NumDestination), nil
}

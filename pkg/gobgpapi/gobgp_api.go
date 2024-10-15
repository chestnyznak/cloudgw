package gobgpapi

import (
	"context"
	"errors"
	"io"
	"time"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

// ConnectToGoBGPAPI connects to GoBGP server by API (gRPC), return API conn with daughter context and cancel func
func ConnectToGoBGPAPI(ctx context.Context, addr string, cancelTimeout int) (bgpapi.GobgpApiClient, context.Context, context.CancelFunc) {
	grpcOpts := []grpc.DialOption{grpc.WithBlock()}

	grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dCtx, cancel := context.WithTimeout(ctx, time.Second*(time.Duration(cancelTimeout)))

	grpcConn, err := grpc.DialContext(dCtx, addr, grpcOpts...)
	if err != nil {
		logger.Fatal("failed to connect to gobgp via grpc", "error", err)
	}

	conn := bgpapi.NewGobgpApiClient(grpcConn)

	return conn, dCtx, cancel
}

// SetGoBGPLocalConfigByAPI creates local GoBGP configuration by API (gRPC)
func SetGoBGPLocalConfigByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, localASN uint32, routerID string, localBGPPort int32) error {
	req := &bgpapi.StartBgpRequest{
		Global: &bgpapi.Global{
			Asn:        localASN,
			RouterId:   routerID,
			ListenPort: localBGPPort,
		},
	}

	_, err := conn.StartBgp(ctx, req)

	return err
}

// GetGoBGPLocalConfigByAPI gets local GoBGP configuration by API (gRPC)
func GetGoBGPLocalConfigByAPI(ctx context.Context, conn bgpapi.GobgpApiClient) (*bgpapi.GetBgpResponse, error) {
	output, err := conn.GetBgp(ctx, &bgpapi.GetBgpRequest{})
	if err != nil {
		return nil, err
	}

	return output, err
}

// AddGoBGPVRFByAPI adds GoBGP VRF by API (gRPC)
func AddGoBGPVRFByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, vrf BGPVRFTable) error {
	vrfTable := &bgpapi.Vrf{
		Name:     vrf.Name,
		Id:       vrf.ID,
		Rd:       vrf.RD,
		ImportRt: vrf.ImportRT,
		ExportRt: vrf.ExportRT,
	}

	_, err := conn.AddVrf(ctx, &bgpapi.AddVrfRequest{
		Vrf: vrfTable,
	})

	return err
}

// DelGoBGPVRFByAPI add GoBGP VRF by API (gRPC)
func DelGoBGPVRFByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, vrfName string) error {
	_, err := conn.DeleteVrf(ctx, &bgpapi.DeleteVrfRequest{
		Name: vrfName,
	})

	return err
}

// GetGoBGPVRFsByAPI gets GoBGP VRF by name by API (gRPC)
func GetGoBGPVRFsByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, vrfName string) (*bgpapi.Vrf, error) {
	response, err := conn.ListVrf(ctx, &bgpapi.ListVrfRequest{
		Name: vrfName,
	})
	if err != nil {
		return nil, err
	}

	r, err := response.Recv()
	if err != nil {
		return nil, err
	}

	return r.Vrf, nil
}

// AddGoBGPPeerByAPI adds BGP Peer on by API (gRPC)
func AddGoBGPPeerByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, peer *BGPPeer) error {
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

	_, err := conn.AddPeer(ctx, &bgpapi.AddPeerRequest{
		Peer: neigh,
	})

	return err
}

// DelGoBGPPeerByAPI deletes BGP peer on by API (gRPC)
func DelGoBGPPeerByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, peerAddress string) error {
	_, err := conn.DeletePeer(ctx, &bgpapi.DeletePeerRequest{
		Address: peerAddress,
	})

	return err
}

// GetGoBGPPeersByAPI gets GoBGP peer list by API (gRPC)
func GetGoBGPPeersByAPI(ctx context.Context, conn bgpapi.GobgpApiClient) ([]*bgpapi.Peer, error) {
	response, err := conn.ListPeer(ctx, &bgpapi.ListPeerRequest{})
	if err != nil {
		return nil, err
	}

	peers := make([]*bgpapi.Peer, 0, 16)

	for {
		r, err := response.Recv()

		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		peers = append(peers, r.Peer)
	}

	return peers, nil
}

// AdvWdrawIPv4PrefixByAPI advertises/withdraws IPv4/Unicast prefix on local GoBGP server
func AdvWdrawIPv4PrefixByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, isAdvertise bool, ipv4BGPNLRIAttrs BGPNLRIAttrs) error {
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
		if _, err := conn.AddPath(ctx, &bgpapi.AddPathRequest{
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
		if _, err := conn.DeletePath(ctx, &bgpapi.DeletePathRequest{
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

// AdvWdrawVPNv4PrefixByAPI advertises/withdraws VPNv4 prefix on local GoBGP server
func AdvWdrawVPNv4PrefixByAPI(ctx context.Context, conn bgpapi.GobgpApiClient, isAdvertise bool, bgpNLRIAttrs BGPNLRIAttrs, sourceASN uint32) error {
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
		TunnelType: 13, // mpls over udp
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
		if _, err := conn.AddPath(ctx, &bgpapi.AddPathRequest{
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
		if _, err := conn.DeletePath(ctx, &bgpapi.DeletePathRequest{
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

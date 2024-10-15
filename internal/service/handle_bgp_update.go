package service

import (
	"context"
	"net"
	"strings"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/interface_types"
	"google.golang.org/protobuf/types/known/anypb"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/pkg/exporter/gobgpexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

const (
	ADVERTISE = true
	WITHDRAW  = false
)

// HandleBGPUpdate watches BGP events (tables and peer state) from GoBGP. The function called once!
func HandleBGPUpdate(
	ctx context.Context,
	vppStream *api.Stream,
	bgpSrv *server.BgpServer,
	cfg config.Config,
	storage *imdb.Storage,
) {
	// map to be used for bgp update parsing to simplify search Update source (tungsten fabric or physical network)
	bgpPeerToPeerTypeMap, err := storage.CreateBGPPeerToTypeMap()
	if err != nil {
		logger.Fatal("failed to get bgp peer to type map", "error", err)
	}

	// map to be used for bgp update parsing to simplify search next-hop

	vppVRFIDToNHMap, err := storage.VPPVRFStorage.CreateVRFIDToNextHopMap()
	if err != nil {
		logger.Fatal("failed to get bgp peer to type map", "error", err)
	}

	// slice to be used for check received address is floating ip or not

	vppAggregatedFIPs := make([]*net.IPNet, 0)

	vrfs := storage.VPPVRFStorage.GetVRFs()

	if vrfs == nil {
		logger.Fatal("failed to get vpp vrfs from memory storage", "logger", err)
	}

	for _, vrf := range vrfs {
		for _, fipPrefix := range vrf.FIPPrefixes {
			_, parsedFIPPrefix, err := net.ParseCIDR(fipPrefix)
			if err != nil {
				logger.Fatal("failed to parse aggregated floating ip prefixes", "logger", err)
			}

			vppAggregatedFIPs = append(vppAggregatedFIPs, parsedFIPPrefix)
		}
	}

	// ========= process bgp updates from tungsten fabric (BEST table) =========================================

	if err := bgpSrv.WatchEvent(ctx, &bgpapi.WatchEventRequest{
		Table: &bgpapi.WatchEventRequest_Table{
			Filters: []*bgpapi.WatchEventRequest_Table_Filter{
				{
					Init: true,
					Type: bgpapi.WatchEventRequest_Table_Filter_BEST, // from BEST table to avoid removing routes when one controller fails
				},
			},
		},
	}, func(r *bgpapi.WatchEventResponse) {
		if t := r.GetTable(); t != nil {
			for _, path := range t.Paths {

				// skip if the update is not from tungsten fabric (it excludes internal updates)

				if storage.BGPPeerStorage.IsConfiguredBGPPeer(path.NeighborIp) && storage.BGPPeerStorage.IsPHYNET(path.NeighborIp) {
					continue
				}

				logger.Debug("got bgp update", "neighbor", path.NeighborIp, "nlri", path.Nlri)

				// parse bgp updates and fill bgpNLRIAttrs structures (except rd/rt)

				fromTF, _, parsedBGPNLRIAttrs, err := ParseBGPUpdate(
					path,
					vppVRFIDToNHMap,
					bgpPeerToPeerTypeMap,
					cfg.TFController.BGPPeerASN,
					cfg.GoBGP.BGPLocalASN,
				)
				if err != nil {
					logger.Error(
						"failed to parse bgp update",
						"path attrs length", len(path.Pattrs),
						"path attrs", pathNLRIString(path.Pattrs),
						"error", err,
					)

					continue
				}

				if !fromTF {
					continue
				}

				// get vrf where the update came from

				calculatedVPPVRF := storage.VPPVRFStorage.GetVRF(parsedBGPNLRIAttrs.VRFID)
				if calculatedVPPVRF == nil {
					logger.Error("vpp vrf not found for received update", "vrf id", parsedBGPNLRIAttrs.VRFID)

					continue
				}

				calculatedBGPVRF := storage.BGPVRFStorage.GetVRF(parsedBGPNLRIAttrs.VRFID)

				if calculatedBGPVRF == nil {
					logger.Error("failed to fetch bgp vrf for received update", "vrf id", parsedBGPNLRIAttrs.VRFID, "error", err)

					continue
				}

				// create vpp ip route structure for floating ip address from parsed bgp update (route with one next-hop as each update has only one next-hop)

				receivedRoute := model.NewVPPIPRoute(
					parsedBGPNLRIAttrs.VRFID,
					interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
					calculatedVPPVRF.SubInterfaceID,
					parsedBGPNLRIAttrs.Prefix,
					[]string{parsedBGPNLRIAttrs.NextHop},
					[]uint32{model.UndefinedTunnelID}, // unknown yet
					[]uint32{parsedBGPNLRIAttrs.MPLSLabel[0]},
				)

				// ========== process update from tungsten fabric with flag WITHDRAW ===========================

				switch path.IsWithdraw {

				case true: // withdraw from tungsten fabric

					// skip update processing if floating ip + next-hop + mpls label does not exist

					if !storage.VPPFIPRouteStorage.IsFIPWithNHAndLabelExist(parsedBGPNLRIAttrs.Prefix, parsedBGPNLRIAttrs.NextHop, parsedBGPNLRIAttrs.MPLSLabel[0]) {
						continue
					}

					// if exists, then find tunnel id and add to vppIPRoute

					storedUDPTunnel := storage.VPPUDPTunnelStorage.GetUDPTunnel(receivedRoute.NextHops[0]) // [0] as the update always has only one nextHop
					if storedUDPTunnel == nil {
						logger.Error("failed to fetch vpp udp tunnel from memory storage", "vrouter", receivedRoute.NextHops[0])

						continue
					}
					receivedRoute.TunnelIDs = []uint32{storedUDPTunnel.TunnelID}

					// get existed floating ip route with its paths (nexthop, mpls, tunnel id)

					storedVPPFIPRoute := storage.VPPFIPRouteStorage.GetFIPRoute(receivedRoute.Prefix)
					if err != nil {
						logger.Error("failed to fetch vpp floating ip route from memory storage", "prefix", receivedRoute.Prefix)

						continue
					}

					// delete existed floating ip and tunnel(s) from vpp and storage

					DelFIPAndTunnelFromVPPAndStorage(
						ctx,
						vppStream,
						bgpSrv,
						cfg,
						*storedVPPFIPRoute,
						storage,
						calculatedVPPVRF,
						calculatedBGPVRF,
					)

					// if stored floating ip had only one path, then finish processing

					if len(storedVPPFIPRoute.NextHops) == 1 {
						continue
					}

					// if 2 or more paths, then remove path of received update from stored floating ip route

					storedVPPFIPRoute.DelPath(receivedRoute.NextHops[0])

					// and then re-create the floating ip route

					AddFIPAndTunnelInVPPAndStorage(
						ctx,
						*vppStream,
						bgpSrv,
						cfg,
						*storedVPPFIPRoute,
						storage,
						calculatedVPPVRF,
						calculatedBGPVRF,
					)

				case false: // advertise from tungsten fabric

					// skip if received prefix is not floating ip to exclude internal cloud addresses handling

					if !netutils.IsFIP(parsedBGPNLRIAttrs.Prefix, vppAggregatedFIPs) {
						continue
					}

					// skip if floating ip + nexthop + mpls label already exists in VPPFIPRouteStorage

					if storage.VPPFIPRouteStorage.IsFIPWithNHAndLabelExist(receivedRoute.Prefix, receivedRoute.NextHops[0], receivedRoute.FIPMPLSLabels[0]) {
						continue
					}

					// find stored floating ip for the received prefix

					storedVPPFIPRoute := storage.VPPFIPRouteStorage.GetFIPRoute(receivedRoute.Prefix)

					// if no stored floating ip found create the floating ip and finish processing

					if storedVPPFIPRoute == nil {
						AddFIPAndTunnelInVPPAndStorage(
							ctx,
							*vppStream,
							bgpSrv,
							cfg,
							receivedRoute,
							storage,
							calculatedVPPVRF,
							calculatedBGPVRF,
						)

						continue
					}

					// if stored floating found, the delete it first from storage and vpp

					DelFIPAndTunnelFromVPPAndStorage(
						ctx,
						vppStream,
						bgpSrv,
						cfg,
						*storedVPPFIPRoute,
						storage,
						calculatedVPPVRF,
						calculatedBGPVRF,
					)

					// then add received path to the stored prefix and then re-create new floating ip

					storedVPPFIPRoute.AddPath(receivedRoute.NextHops[0], receivedRoute.TunnelIDs[0], receivedRoute.FIPMPLSLabels[0])

					AddFIPAndTunnelInVPPAndStorage(
						ctx,
						*vppStream,
						bgpSrv,
						cfg,
						*storedVPPFIPRoute,
						storage,
						calculatedVPPVRF,
						calculatedBGPVRF,
					)
				}
			}
		}
	}); err != nil {
		logger.Error("failed to handle event response for table update from tungsten fabric", "error", err)
	}

	// ========== processing bgp updates from physical network ==============================================

	if err := bgpSrv.WatchEvent(ctx, &bgpapi.WatchEventRequest{
		Table: &bgpapi.WatchEventRequest_Table{
			Filters: []*bgpapi.WatchEventRequest_Table_Filter{
				{
					Init: true,
					Type: bgpapi.WatchEventRequest_Table_Filter_POST_POLICY, // use POST_POLICY as updates from cloudgw to tungsten fabrics are self-generated and not removed from BEST table
				},
			},
		},
	}, func(r *bgpapi.WatchEventResponse) {
		if t := r.GetTable(); t != nil {
			for _, path := range t.Paths {

				// skip if the update is not from physical network

				if storage.BGPPeerStorage.IsConfiguredBGPPeer(path.NeighborIp) && storage.BGPPeerStorage.IsTF(path.NeighborIp) {
					continue
				}
				logger.Debug("got update", "neighbor", path.NeighborIp, "nlri", path.Nlri)

				// parse bgp updates and fill only parsed attributes in bgpNLRIAttrs structure

				_, fromPN, parsedBGPNLRIAttrs, err := ParseBGPUpdate(
					path,
					vppVRFIDToNHMap,
					bgpPeerToPeerTypeMap,
					cfg.TFController.BGPPeerASN,
					cfg.GoBGP.BGPLocalASN,
				)
				if err != nil {
					logger.Error("failed to parse bgp update", "path attrs length", len(path.Pattrs), "path attrs", pathNLRIString(path.Pattrs), "error", err)
				}

				if !fromPN {
					continue
				}

				// get vrf where from the update

				calculatedVPPVRF := storage.VPPVRFStorage.GetVRF(parsedBGPNLRIAttrs.VRFID)

				calculatedBGPVRF := storage.BGPVRFStorage.GetVRF(parsedBGPNLRIAttrs.VRFID)

				if calculatedBGPVRF == nil { //nolint:staticcheck
					logger.Error("failed to fetch bgp vrf", "vrf id", parsedBGPNLRIAttrs.VRFID, "error", err)
				}

				// get default vrf

				defaultVPPVRF := storage.VPPVRFStorage.GetVRF(0)

				// create attributes of vpp ip route to physical network to be installed/removed on specific vpp's vrf

				vppIPRoute := model.NewVPPIPRoute(
					parsedBGPNLRIAttrs.VRFID,
					interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
					calculatedVPPVRF.SubInterfaceID,
					parsedBGPNLRIAttrs.Prefix,
					[]string{parsedBGPNLRIAttrs.NextHop},
					nil,
					nil,
				)

				// create bgp nlri attributes for selected aggregated prefix to be advertised/withdraw to/from tungsten fabric

				aggrNLRIAttr := gobgpapi.NewBGPNLRIAttrs(
					parsedBGPNLRIAttrs.Prefix,
					defaultVPPVRF.LocalAddr, // as the VPP handles the traffic from vRouters
					0,
					calculatedBGPVRF.RD,       //nolint:staticcheck
					calculatedBGPVRF.ExportRT, // tungsten fabric must read the rt and import in local vrf
					[]uint32{calculatedVPPVRF.MPLSLocalLabel},
				)

				switch path.IsWithdraw {

				case true: // withdraw route from physical network

					if err = vpp.AddDelIPRoute(*vppStream, false, vppIPRoute); err != nil {
						logger.Error("failed to delete ip route", "prefix", vppIPRoute.Prefix, "error", err)
					}

					// withdraw (delete) physical network's enriched prefix from tungsten fabric

					if err = gobgp.AdvWdrawVpnv4Prefix(
						ctx,
						bgpSrv,
						WITHDRAW,
						aggrNLRIAttr,
						calculatedBGPVRF.PeerASN, // need for loop prevention
					); err != nil {
						logger.Error("failed to withdraw vpnv4 prefix", "prefix", aggrNLRIAttr.Prefix, "error", err)
					}

				case false: // advertise route from physical network

					// create new upstream ipv4 route through physical network

					if err = vpp.AddDelIPRoute(*vppStream, true, vppIPRoute); err != nil {
						logger.Error("failed to run add ip route", "prefix", vppIPRoute.Prefix, "error", err)
					}

					// advertise the enriched route from physical network to tungsten fabric with local assigned mpls Label and rt (should match tungsten fabric virtual network settings)

					if err = gobgp.AdvWdrawVpnv4Prefix(
						ctx,
						bgpSrv,
						ADVERTISE,
						aggrNLRIAttr,
						calculatedBGPVRF.PeerASN, // need for loop prevention
					); err != nil {
						logger.Error("failed to advertise vpnv4 prefix", "prefix", aggrNLRIAttr.Prefix, "error", err)

						continue
					}
				}
			}
		}
	}); err != nil {
		logger.Error("failed to handle event response for table update from physical network", "error", err)
	}

	// ========= processing bgp peers status change as events =====================================

	if err := bgpSrv.WatchEvent(ctx, &bgpapi.WatchEventRequest{Peer: &bgpapi.WatchEventRequest_Peer{}}, func(r *bgpapi.WatchEventResponse) {
		if peer := r.GetPeer(); peer != nil {
			peerIP := peer.Peer.State.NeighborAddress

			if peerIP == "<nil>" {
				return
			}

			bgpPeer := storage.BGPPeerStorage.GetBGPPeer(peerIP)

			if bgpPeer == nil { //nolint:staticcheck
				logger.Error("failed to find bgp peer for ip", "peer ip", peerIP)
			}

			// update bgp state in BGPPeerStorage

			currentState := bgpPeer.BGPPeerState //nolint:staticcheck

			if err = storage.UpdateBGPPeerState(peerIP, currentState, peer.Peer.State.SessionState); err != nil {
				logger.Error("failed to update bgp peer state", "error", err)
			}

			// update gobgp metrics

			if bgpPeer.BGPPeerState == bgpapi.PeerState_ESTABLISHED { // changed from DOWN to UP
				gobgpexporter.GoBGPGeneralMetrics.IncActivePeerCount()
			}

			if bgpPeer.BGPPeerPrevState == bgpapi.PeerState_ESTABLISHED { // changed from UP to DOWN
				gobgpexporter.GoBGPGeneralMetrics.DecActivePeerCount()
			}

			// start bfd monitoring when bgp peer state changed to ESTABLISHED

			if bgpPeer.PeerType == model.PHYNET && bgpPeer.BFDPeering.BFDEnabled {
				if bgpPeer.BGPPeerState == bgpapi.PeerState_ESTABLISHED && !bgpPeer.BFDPeering.BFDPeerEstablished {
					go CheckBFDPeerStatus(ctx, *bgpPeer)

					storage.UpdateBFDPeerState(peerIP, true)

					logger.Info("start setting up a bfd session", "peer address", peerIP)
				}
			}
		}
	}); err != nil {
		logger.Fatal("failed to handle event response for bgp peer events", "error", err)
	}
}

func pathNLRIString(pathAttrs []*anypb.Any) string {
	tmp := make([]string, len(pathAttrs))

	for _, pathAttr := range pathAttrs {
		tmp = append(tmp, pathAttr.String())
	}

	return strings.Join(tmp, ", ")
}

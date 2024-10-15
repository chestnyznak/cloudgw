package test_test

import (
	"testing"
	"time"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/suite"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
)

var (
	fipFixtures = []*model.VPPIPRoute{
		{VRFID: 1, Prefix: "10.11.64.1/32", NextHops: []string{"10.0.0.1", "10.0.0.2"}, MainInterfaceID: 1, SubInterfaceID: 2, TunnelIDs: []uint32{1, 2}, FIPMPLSLabels: []uint32{10, 20}},
		{VRFID: 1, Prefix: "10.11.64.2/32", NextHops: []string{"10.0.0.2"}, MainInterfaceID: 1, SubInterfaceID: 2, TunnelIDs: []uint32{2}, FIPMPLSLabels: []uint32{20}},
		{VRFID: 1, Prefix: "10.11.64.3/32", NextHops: []string{"10.0.0.2"}, MainInterfaceID: 1, SubInterfaceID: 2, TunnelIDs: []uint32{2}, FIPMPLSLabels: []uint32{30}},
		{VRFID: 1, Prefix: "10.11.64.4/32", NextHops: []string{"10.0.0.3"}, MainInterfaceID: 1, SubInterfaceID: 2, TunnelIDs: []uint32{3}, FIPMPLSLabels: []uint32{40}},
		{VRFID: 1, Prefix: "10.11.64.5/32", NextHops: []string{"10.0.0.1", "10.0.0.2, 10.0.0.4"}, MainInterfaceID: 1, SubInterfaceID: 2, TunnelIDs: []uint32{1, 2, 4}, FIPMPLSLabels: []uint32{10, 20, 50}},
	}
	udpTunnelFixtures = []*model.VPPUDPTunnel{
		{RoutingTableID: 1, TunnelID: 1, SrcIP: "10.0.0.1", DstIP: "10.10.10.1", SrcPort: 50000, DstPort: 6635, FIPServed: 0},
		{RoutingTableID: 1, TunnelID: 2, SrcIP: "10.0.0.1", DstIP: "10.10.10.2", SrcPort: 50002, DstPort: 6635, FIPServed: 10},
		{RoutingTableID: 1, TunnelID: 3, SrcIP: "10.0.0.1", DstIP: "10.10.10.3", SrcPort: 50003, DstPort: 6635, FIPServed: 20},
		{RoutingTableID: 1, TunnelID: 4, SrcIP: "10.0.0.1", DstIP: "10.10.10.4", SrcPort: 50004, DstPort: 6635, FIPServed: 30},
		{RoutingTableID: 1, TunnelID: 5, SrcIP: "10.0.0.1", DstIP: "10.10.10.5", SrcPort: 50005, DstPort: 6635, FIPServed: 40},
	}
	bgpPeerFixtures = []*model.BGPPeer{
		{PeerType: model.TF, PeerASN: 65000, PeerAddress: "10.0.0.1", PeerPort: 169, Md5Password: "", EbgpMultiHop: false, EbgpMultiHopTTL: 1, VRFName: "", AFI: bgpapi.Family_AFI_IP, SAFI: bgpapi.Family_SAFI_MPLS_VPN, KeepAliveTimer: 3, HoldTimer: 9, BGPPeerState: bgpapi.PeerState_UNKNOWN, BGPPeerPrevState: bgpapi.PeerState_UNKNOWN, BGPPeerLastActivity: time.Now(), BFDPeering: &model.BFDPeer{BFDPeerEstablished: false}},
		{PeerType: model.TF, PeerASN: 65000, PeerAddress: "10.0.0.2", PeerPort: 169, Md5Password: "", EbgpMultiHop: false, EbgpMultiHopTTL: 1, VRFName: "", AFI: bgpapi.Family_AFI_IP, SAFI: bgpapi.Family_SAFI_MPLS_VPN, KeepAliveTimer: 3, HoldTimer: 9, BGPPeerState: bgpapi.PeerState_UNKNOWN, BGPPeerPrevState: bgpapi.PeerState_UNKNOWN, BGPPeerLastActivity: time.Now(), BFDPeering: &model.BFDPeer{BFDPeerEstablished: false}},
		{PeerType: model.TF, PeerASN: 65000, PeerAddress: "10.0.0.3", PeerPort: 169, Md5Password: "", EbgpMultiHop: false, EbgpMultiHopTTL: 1, VRFName: "", AFI: bgpapi.Family_AFI_IP, SAFI: bgpapi.Family_SAFI_MPLS_VPN, KeepAliveTimer: 3, HoldTimer: 9, BGPPeerState: bgpapi.PeerState_UNKNOWN, BGPPeerPrevState: bgpapi.PeerState_UNKNOWN, BGPPeerLastActivity: time.Now(), BFDPeering: &model.BFDPeer{BFDPeerEstablished: false}},
		{PeerType: model.PHYNET, PeerASN: 64555, PeerAddress: "10.1.1.1", PeerPort: 169, Md5Password: "", EbgpMultiHop: false, EbgpMultiHopTTL: 1, VRFName: "test", AFI: bgpapi.Family_AFI_IP, SAFI: bgpapi.Family_SAFI_UNICAST, KeepAliveTimer: 3, HoldTimer: 9, BGPPeerState: bgpapi.PeerState_UNKNOWN, BGPPeerPrevState: bgpapi.PeerState_UNKNOWN, BGPPeerLastActivity: time.Now(), BFDPeering: &model.BFDPeer{BFDPeerEstablished: false}},
		{PeerType: model.PHYNET, PeerASN: 64555, PeerAddress: "10.1.1.2", PeerPort: 169, Md5Password: "", EbgpMultiHop: false, EbgpMultiHopTTL: 1, VRFName: "test", AFI: bgpapi.Family_AFI_IP, SAFI: bgpapi.Family_SAFI_UNICAST, KeepAliveTimer: 3, HoldTimer: 9, BGPPeerState: bgpapi.PeerState_UNKNOWN, BGPPeerPrevState: bgpapi.PeerState_UNKNOWN, BGPPeerLastActivity: time.Now(), BFDPeering: &model.BFDPeer{BFDPeerEstablished: false}},
	}
	bgpVRFFixtures = []*model.BGPVRFTable{
		{Name: "test01", ID: 1, LocalASN: 65000, PeerASN: 64555, RD: nil, ExportRT: nil, ImportRT: nil},
		{Name: "test02", ID: 2, LocalASN: 65000, PeerASN: 64555, RD: nil, ExportRT: nil, ImportRT: nil},
		{Name: "test03", ID: 3, LocalASN: 65000, PeerASN: 64555, RD: nil, ExportRT: nil, ImportRT: nil},
	}
	vppVRFFixtures = []*model.VPPVRFTable{
		{Name: "test01", ID: 0, MainInterfaceID: 1, SubInterfaceID: 2, VLAN: 100, LocalAddr: "10.0.1.1", LocalAddrLen: 24, NextHop: "10.0.1.254", MPLSLocalLabel: 1000, FIPPrefixes: []string{"192.1.0.0/24", "192.1.1.0/24"}, FIPServed: 0},
		{Name: "test02", ID: 1, MainInterfaceID: 1, SubInterfaceID: 3, VLAN: 200, LocalAddr: "10.0.2.1", LocalAddrLen: 24, NextHop: "10.0.2.254", MPLSLocalLabel: 2000, FIPPrefixes: []string{"192.2.0.0/24", "192.2.1.0/24"}, FIPServed: 0},
		{Name: "test03", ID: 2, MainInterfaceID: 1, SubInterfaceID: 4, VLAN: 300, LocalAddr: "10.0.3.1", LocalAddrLen: 24, NextHop: "10.0.3.254", MPLSLocalLabel: 3000, FIPPrefixes: []string{"192.3.0.0/24", "192.3.1.0/24"}, FIPServed: 0},
	}
)

type IMDBStorageSuite struct {
	suite.Suite
	fipRouteStorage  imdb.VPPFIPRouteStorage
	udpTunnelStorage imdb.VPPUDPTunnelStorage
	bgpPeerStorage   imdb.BGPPeerStorage
	bgpVRFStorage    imdb.BGPVRFStorage
	vppVRFStorage    imdb.VPPVRFStorage
}

func TestVPPFIPRouteStorage(t *testing.T) {
	suite.Run(t, new(IMDBStorageSuite))
}

func (s *IMDBStorageSuite) SetupSuite() {
	fipRouteStorage := imdb.NewVPPFIPRouteStorage()
	s.fipRouteStorage = *fipRouteStorage

	udpTunnelStorage := imdb.NewVPPUDPTunnelStorage()
	s.udpTunnelStorage = *udpTunnelStorage

	bgpPeerStorage := imdb.NewBGPPeerStorage()
	s.bgpPeerStorage = *bgpPeerStorage

	bgpVRFStorage := imdb.NewBGPVRFStorage()
	s.bgpVRFStorage = *bgpVRFStorage

	vppVRFStorage := imdb.NewVPPVRFStorage()
	s.vppVRFStorage = *vppVRFStorage
}

func (s *IMDBStorageSuite) SetupTest() {
	for _, fip := range fipFixtures {
		err := s.fipRouteStorage.AddFIPRoute(fip)
		s.Require().NoError(err)
	}

	for _, tunnel := range udpTunnelFixtures {
		err := s.udpTunnelStorage.AddUDPTunnel(tunnel)
		s.Require().NoError(err)
	}

	for _, peer := range bgpPeerFixtures {
		err := s.bgpPeerStorage.AddBGPPeer(peer)
		s.Require().NoError(err)
	}

	for _, vrf := range bgpVRFFixtures {
		err := s.bgpVRFStorage.AddVRF(vrf)
		s.Require().NoError(err)
	}

	for _, vrf := range vppVRFFixtures {
		err := s.vppVRFStorage.AddVRF(vrf)
		s.Require().NoError(err)
	}
}

func (s *IMDBStorageSuite) TearDownTest() {
	err := s.fipRouteStorage.DelFIPRoutes()
	s.Require().NoError(err)

	err = s.udpTunnelStorage.DelUDPTunnels()
	s.Require().NoError(err)

	err = s.bgpPeerStorage.DelBGPPeers()
	s.Require().NoError(err)

	err = s.bgpVRFStorage.DelVRFs()
	s.Require().NoError(err)

	err = s.vppVRFStorage.DelVRFs()
	s.Require().NoError(err)
}

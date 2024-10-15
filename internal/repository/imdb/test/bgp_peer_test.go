package test_test

import (
	bgpapi "github.com/osrg/gobgp/v3/api"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
)

func (s *IMDBStorageSuite) TestGetBGPPeer() {
	peer := s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(peer.PeerAddress, "10.0.0.1")
	s.Require().Equal(peer.PeerType, model.TF)
	s.Require().NotEqual(peer.PeerType, model.PHYNET)

	peer = s.bgpPeerStorage.GetBGPPeer("10.1.1.1")
	s.Require().Equal(peer.PeerAddress, "10.1.1.1")
	s.Require().Equal(peer.PeerType, model.PHYNET)
	s.Require().NotEqual(peer.PeerType, model.TF)

	peer = s.bgpPeerStorage.GetBGPPeer("10.0.0.0")
	s.Require().Nil(peer)

	peer = s.bgpPeerStorage.GetBGPPeer("")
	s.Require().Nil(peer)
}

func (s *IMDBStorageSuite) TestGetBGPPeers() {
	peers := s.bgpPeerStorage.GetBGPPeers()
	s.Require().Equal(len(bgpPeerFixtures), len(peers))

	err := s.bgpPeerStorage.DelBGPPeers()
	s.Require().NoError(err)

	peers = s.bgpPeerStorage.GetBGPPeers()
	s.Require().Nil(peers)
}

func (s *IMDBStorageSuite) TestIsConfiguredBGPPeer() {
	isConfigured := s.bgpPeerStorage.IsConfiguredBGPPeer("10.0.0.1")
	s.Require().True(isConfigured)

	isConfigured = s.bgpPeerStorage.IsConfiguredBGPPeer("10.1.1.1")
	s.Require().True(isConfigured)

	isConfigured = s.bgpPeerStorage.IsConfiguredBGPPeer("10.0.0.0")
	s.Require().False(isConfigured)

	isConfigured = s.bgpPeerStorage.IsConfiguredBGPPeer("")
	s.Require().False(isConfigured)
}

func (s *IMDBStorageSuite) TestCreateBGPPeerToTypeMap() {
	peerTypeMap, err := s.bgpPeerStorage.CreateBGPPeerToTypeMap()
	s.Require().NoError(err)
	s.Require().Equal(len(bgpPeerFixtures), len(peerTypeMap))

	for i := 0; i < len(bgpPeerFixtures); i++ {
		s.Require().Equal(bgpPeerFixtures[i].PeerType, peerTypeMap[bgpPeerFixtures[i].PeerAddress])
	}
}

func (s *IMDBStorageSuite) TestIsTFIsPHYNET() {
	for _, peer := range bgpPeerFixtures {
		isPN := s.bgpPeerStorage.IsPHYNET(peer.PeerAddress)
		isTF := s.bgpPeerStorage.IsTF(peer.PeerAddress)

		if peer.PeerType == model.PHYNET {
			s.Require().True(isPN)
			s.Require().False(isTF)
		} else {
			s.Require().False(isPN)
			s.Require().True(isTF)
		}
	}
}

func (s *IMDBStorageSuite) TestUpdateBGPPeerState() {
	peer := s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(bgpapi.PeerState_UNKNOWN, peer.BGPPeerPrevState)
	s.Require().Equal(bgpapi.PeerState_UNKNOWN, peer.BGPPeerState)

	err := s.bgpPeerStorage.UpdateBGPPeerState("10.0.0.1", bgpapi.PeerState_ACTIVE, bgpapi.PeerState_ESTABLISHED)
	s.Require().NoError(err)

	peer = s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(bgpapi.PeerState_ACTIVE, peer.BGPPeerPrevState)
	s.Require().Equal(bgpapi.PeerState_ESTABLISHED, peer.BGPPeerState)

	err = s.bgpPeerStorage.UpdateBGPPeerState("10.0.0.0", bgpapi.PeerState_ACTIVE, bgpapi.PeerState_ESTABLISHED)
	s.Require().ErrorIs(err, imdb.ErrNoBGPPeerFoundInStorage)

	err = s.bgpPeerStorage.UpdateBGPPeerState("", bgpapi.PeerState_ACTIVE, bgpapi.PeerState_ESTABLISHED)
	s.Require().ErrorIs(err, imdb.ErrNoBGPPeerFoundInStorage)
}

func (s *IMDBStorageSuite) TestUpdateBFDPeerState() {
	peer := s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(false, peer.BFDPeering.BFDPeerEstablished)

	s.bgpPeerStorage.UpdateBFDPeerState("10.0.0.1", true)

	peer = s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(true, peer.BFDPeering.BFDPeerEstablished)

	s.bgpPeerStorage.UpdateBFDPeerState("10.0.0.1", false)

	peer = s.bgpPeerStorage.GetBGPPeer("10.0.0.1")
	s.Require().Equal(false, peer.BFDPeering.BFDPeerEstablished)

	// Check no panic
	s.bgpPeerStorage.UpdateBFDPeerState("", true)
	s.bgpPeerStorage.UpdateBFDPeerState("", false)
}

package test_test

import (
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
)

func (s *IMDBStorageSuite) TestDelUDPTunnel() {
	err := s.udpTunnelStorage.DelUDPTunnel("10.10.10.1")
	s.Require().NoError(err)

	err = s.udpTunnelStorage.DelUDPTunnel("10.10.10.2")
	s.Require().NoError(err)

	err = s.udpTunnelStorage.DelUDPTunnel("10.10.10.2")
	s.Require().ErrorIs(err, imdb.ErrNoVPPUDPTunnelFoundInStorage)

	err = s.udpTunnelStorage.DelUDPTunnel("10.10.10.")
	s.Require().ErrorIs(err, imdb.ErrNoVPPUDPTunnelFoundInStorage)
}

func (s *IMDBStorageSuite) TestGetUDPTunnel() {
	tunnel := s.udpTunnelStorage.GetUDPTunnel("10.10.10.1")
	s.Require().Equal(tunnel.DstIP, "10.10.10.1")
	s.Require().Equal(tunnel.SrcIP, "10.0.0.1")

	tunnel = s.udpTunnelStorage.GetUDPTunnel("10.10.10.0")
	s.Require().Nil(tunnel)

	tunnel = s.udpTunnelStorage.GetUDPTunnel("")
	s.Require().Nil(tunnel)

	tunnel = s.udpTunnelStorage.GetUDPTunnel("10.0.0.1")
	s.Require().Nil(tunnel)
}

func (s *IMDBStorageSuite) TestGetUDPTunnels() {
	tunnels := s.udpTunnelStorage.GetUDPTunnels()
	s.Require().Equal(len(udpTunnelFixtures), len(tunnels))

	err := s.udpTunnelStorage.DelUDPTunnels()
	s.Require().NoError(err)

	tunnels = s.udpTunnelStorage.GetUDPTunnels()
	s.Require().Nil(tunnels)
}

func (s *IMDBStorageSuite) TestIncDecFIPServed() {
	s.udpTunnelStorage.IncFIPServed("10.10.10.1")
	s.Require().Equal(uint32(1), s.udpTunnelStorage.GetFIPServed("10.10.10.1"))

	s.udpTunnelStorage.DecFIPServed("10.10.10.1")
	s.Require().Equal(uint32(0), s.udpTunnelStorage.GetFIPServed("10.10.10.1"))

	s.udpTunnelStorage.DecFIPServed("10.10.10.1")
	s.Require().Equal(uint32(0), s.udpTunnelStorage.GetFIPServed("10.10.10.1"))

	s.udpTunnelStorage.IncFIPServed("10.10.10.2")
	s.Require().Equal(uint32(11), s.udpTunnelStorage.GetFIPServed("10.10.10.2"))

	s.udpTunnelStorage.DecFIPServed("10.10.10.2")
	s.Require().Equal(uint32(10), s.udpTunnelStorage.GetFIPServed("10.10.10.2"))

	// Check no panic
	s.udpTunnelStorage.IncFIPServed("")
	s.udpTunnelStorage.DecFIPServed("")
	s.udpTunnelStorage.IncFIPServed("1.1.1.1")
	s.udpTunnelStorage.DecFIPServed("1.1.1.1")
}

func (s *IMDBStorageSuite) TestIsUDPTunnelExist() {
	isExist := s.udpTunnelStorage.IsUDPTunnelExist("10.10.10.1")
	s.Require().True(isExist)

	isExist = s.udpTunnelStorage.IsUDPTunnelExist("10.10.10.2")
	s.Require().True(isExist)

	isExist = s.udpTunnelStorage.IsUDPTunnelExist("10.10.10.0")
	s.Require().False(isExist)

	isExist = s.udpTunnelStorage.IsUDPTunnelExist("10.10.10.11")
	s.Require().False(isExist)

	isExist = s.udpTunnelStorage.IsUDPTunnelExist("")
	s.Require().False(isExist)
}

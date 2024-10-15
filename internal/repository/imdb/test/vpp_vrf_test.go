package test_test

func (s *IMDBStorageSuite) TestGetVPPVRF() {
	vrfs := s.vppVRFStorage.GetVRFs()
	s.Require().Equal(len(vppVRFFixtures), len(vrfs))

	for i := 0; i < len(vrfs); i++ {
		s.Require().Equal(vppVRFFixtures[i], vrfs[i])
	}
}

func (s *IMDBStorageSuite) TestIncDecVPPFIPServed() {
	s.vppVRFStorage.IncFIPServed(1)
	s.Require().Equal(uint32(1), s.vppVRFStorage.GetFIPServed(1))

	s.vppVRFStorage.DecFIPServed(1)
	s.Require().Equal(uint32(0), s.vppVRFStorage.GetFIPServed(1))

	s.vppVRFStorage.DecFIPServed(1)
	s.Require().Equal(uint32(0), s.vppVRFStorage.GetFIPServed(1))

	s.vppVRFStorage.IncFIPServed(2)
	s.Require().Equal(uint32(1), s.vppVRFStorage.GetFIPServed(2))

	s.vppVRFStorage.DecFIPServed(2)
	s.Require().Equal(uint32(0), s.vppVRFStorage.GetFIPServed(2))

	// Check no panic
	s.vppVRFStorage.IncFIPServed(100)
}

func (s *IMDBStorageSuite) TestGetVRF() {
	vrf := s.vppVRFStorage.GetVRF(0)
	s.Require().NotNil(vrf)
	s.Require().Equal(vppVRFFixtures[0].LocalAddr, vrf.LocalAddr)

	vrf = s.vppVRFStorage.GetVRF(1)
	s.Require().NotNil(vrf)
	s.Require().Equal(vppVRFFixtures[1].LocalAddr, vrf.LocalAddr)

	vrf = s.vppVRFStorage.GetVRF(100)
	s.Require().Nil(vrf)
}

func (s *IMDBStorageSuite) TestIsVRFExist() {
	isExist := s.vppVRFStorage.IsVRFExist(0)
	s.Require().True(isExist)

	isExist = s.vppVRFStorage.IsVRFExist(1)
	s.Require().True(isExist)

	isExist = s.vppVRFStorage.IsVRFExist(2)
	s.Require().True(isExist)

	isExist = s.vppVRFStorage.IsVRFExist(100)
	s.Require().False(isExist)
}

func (s *IMDBStorageSuite) TestCreateVRFIDToNextHopMap() {
	vrfMap, err := s.vppVRFStorage.CreateVRFIDToNextHopMap()
	s.Require().NoError(err)
	s.Require().Equal(len(vppVRFFixtures), len(vrfMap))

	for i, vrf := range vppVRFFixtures {
		s.Require().Equal(vrf.NextHop, vrfMap[vppVRFFixtures[i].ID])
	}
}

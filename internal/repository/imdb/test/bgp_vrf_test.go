package test_test

func (s *IMDBStorageSuite) TestGetBGPVRF() {
	vrf := s.bgpVRFStorage.GetVRF(1)
	s.Require().Equal(vrf.Name, "test01")

	vrf = s.bgpVRFStorage.GetVRF(2)
	s.Require().Equal(vrf.Name, "test02")

	vrf = s.bgpVRFStorage.GetVRF(0)
	s.Require().Nil(vrf)

	vrf = s.bgpVRFStorage.GetVRF(100)
	s.Require().Nil(vrf)
}

func (s *IMDBStorageSuite) TestGetVRFs() {
	vrfs := s.bgpVRFStorage.GetVRFs()
	s.Require().Equal(len(bgpVRFFixtures), len(vrfs))

	err := s.bgpVRFStorage.DelVRFs()
	s.Require().NoError(err)

	vrfs = s.bgpVRFStorage.GetVRFs()
	s.Require().Nil(vrfs)
}

package test_test

import (
	"fmt"
	"log"
	"sync"
	"testing"

	"go.fd.io/govpp/binapi/interface_types"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
)

func (s *IMDBStorageSuite) TestGetFIPRoute() {
	fip := s.fipRouteStorage.GetFIPRoute("10.11.64.1/32")
	s.Require().Equal(fip.Prefix, "10.11.64.1/32")
	s.Require().Equal(fip.NextHops, []string{"10.0.0.1", "10.0.0.2"})

	fip = s.fipRouteStorage.GetFIPRoute("10.11.64.1")
	s.Require().Nil(fip)

	fip = s.fipRouteStorage.GetFIPRoute("")
	s.Require().Nil(fip)
}

func (s *IMDBStorageSuite) TestGetFIPRoutes() {
	fips := s.fipRouteStorage.GetFIPRoutes()
	s.Require().Equal(len(fipFixtures), len(fips))

	err := s.fipRouteStorage.DelFIPRoutes()
	s.Require().NoError(err)

	fips = s.fipRouteStorage.GetFIPRoutes()
	s.Require().Nil(fips)
}

func (s *IMDBStorageSuite) TestIsFIPPrefixExist() {
	isExist := s.fipRouteStorage.IsFIPPrefixExist("10.11.64.1/32")
	s.Require().True(isExist)

	isExist = s.fipRouteStorage.IsFIPPrefixExist("10.11.64.2/32")
	s.Require().True(isExist)

	isExist = s.fipRouteStorage.IsFIPPrefixExist("10.11.64.2")
	s.Require().False(isExist)

	isExist = s.fipRouteStorage.IsFIPPrefixExist("")
	s.Require().False(isExist)
}

func (s *IMDBStorageSuite) TestIsFIPWithNHAndLabelExist() {
	isExist := s.fipRouteStorage.IsFIPWithNHAndLabelExist("10.11.64.1/32", "10.0.0.1", 10)
	s.Require().True(isExist)

	isExist = s.fipRouteStorage.IsFIPWithNHAndLabelExist("10.11.64.1/32", "10.0.0.2", 20)
	s.Require().True(isExist)

	isExist = s.fipRouteStorage.IsFIPWithNHAndLabelExist("10.11.64.1/32", "10.0.0.3", 10)
	s.Require().False(isExist)

	isExist = s.fipRouteStorage.IsFIPWithNHAndLabelExist("10.11.64.1/32", "", 10)
	s.Require().False(isExist)

	isExist = s.fipRouteStorage.IsFIPWithNHAndLabelExist("", "10.0.0.3", 10)
	s.Require().False(isExist)

	isExist = s.fipRouteStorage.IsFIPWithNHAndLabelExist("", "", 0)
	s.Require().False(isExist)
}

func (s *IMDBStorageSuite) TestAddFIPRouteConcurrency() {
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		var err error

		for i := 0; i < 100000; i++ {
			fip := model.VPPIPRoute{
				VRFID:           1,
				Prefix:          fmt.Sprintf("10.11.65.%d/32", i),
				NextHops:        []string{fmt.Sprintf("10.0.0.%d/32", i)},
				MainInterfaceID: 1,
				SubInterfaceID:  1,
				TunnelIDs:       []uint32{uint32(i)},
				FIPMPLSLabels:   []uint32{uint32(i * 10)},
			}
			err = s.fipRouteStorage.AddFIPRoute(&fip)
			s.Require().NoError(err)
		}
	}()

	go func() {
		defer wg.Done()

		var err error

		for i := 0; i < 100000; i++ {
			fip := model.VPPIPRoute{
				VRFID:           1,
				Prefix:          fmt.Sprintf("10.11.66.%d/32", i),
				NextHops:        []string{fmt.Sprintf("10.0.0.%d/32", i)},
				MainInterfaceID: 1,
				SubInterfaceID:  1,
				TunnelIDs:       []uint32{uint32(i)},
				FIPMPLSLabels:   []uint32{uint32(i * 10)},
			}
			err = s.fipRouteStorage.AddFIPRoute(&fip)
			s.Require().NoError(err)
		}
	}()

	wg.Wait()

	fips := s.fipRouteStorage.GetFIPRoutes()
	s.Require().Equal(200000+len(fipFixtures), len(fips))
}

func (s *IMDBStorageSuite) TestDelFIPRoute() {
	err := s.fipRouteStorage.DelFIPRoute("10.11.64.1/32")
	s.Require().NoError(err)

	err = s.fipRouteStorage.DelFIPRoute("10.11.64.1/32")
	s.Require().ErrorIs(err, imdb.ErrNoVPPFIPFoundInStorage)

	err = s.fipRouteStorage.DelFIPRoute("10.11.64.2/32")
	s.Require().NoError(err)

	err = s.fipRouteStorage.DelFIPRoute("10.11.64.1")
	s.Require().ErrorIs(err, imdb.ErrNoVPPFIPFoundInStorage)

	err = s.fipRouteStorage.DelFIPRoute("10.11.64.3")
	s.Require().ErrorIs(err, imdb.ErrNoVPPFIPFoundInStorage)

	err = s.fipRouteStorage.DelFIPRoute("")
	s.Require().ErrorIs(err, imdb.ErrNoVPPFIPFoundInStorage)
}

func BenchmarkPureMapStorage(_ *testing.B) {
	count := 10000
	storage := make(map[string]model.VPPIPRoute)

	var mu sync.Mutex

	for i := 0; i < count; i++ {
		mu.Lock()
		storage[fmt.Sprintf("%d", i)] = model.VPPIPRoute{
			VRFID:           1,
			MainInterfaceID: interface_types.InterfaceIndex(1),
			SubInterfaceID:  interface_types.InterfaceIndex(2),
			Prefix:          fmt.Sprintf("%d", i),
			NextHops:        []string{fmt.Sprintf("%d", i)},
			TunnelIDs:       []uint32{uint32(i)},
			FIPMPLSLabels:   []uint32{uint32(i)},
		}
		mu.Unlock()
	}

	for i := 0; i < count; i++ {
		mu.Lock()
		_ = storage[fmt.Sprintf("%d", i)]
		mu.Unlock()
	}
}

func BenchmarkIMDBStorage(_ *testing.B) {
	count := 10000

	storage := imdb.NewVPPFIPRouteStorage()

	for i := 0; i < count; i++ {
		fip := model.VPPIPRoute{
			VRFID:           1,
			MainInterfaceID: interface_types.InterfaceIndex(1),
			SubInterfaceID:  interface_types.InterfaceIndex(2),
			Prefix:          fmt.Sprintf("%d", i),
			NextHops:        []string{fmt.Sprintf("%d", i)},
			TunnelIDs:       []uint32{uint32(i)},
			FIPMPLSLabels:   []uint32{uint32(i)},
		}

		err := storage.AddFIPRoute(&fip)
		if err != nil {
			log.Fatal(err)
		}
	}

	for i := 0; i < count; i++ {
		fip := storage.GetFIPRoute(fmt.Sprintf("%d", i))
		if fip == nil {
			log.Fatal("floating ip is nil")
		}
	}
}

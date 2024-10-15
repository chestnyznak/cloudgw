package imdb

import (
	"github.com/hashicorp/go-memdb"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var VPPFIPRouteTableName = "fip_route"

type VPPFIPRouteStorage struct {
	db *memdb.MemDB
}

func NewVPPFIPRouteStorage() *VPPFIPRouteStorage {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			VPPFIPRouteTableName: {
				Name: VPPFIPRouteTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Prefix"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		logger.Fatal("failed to create vpp floating ip route storage", "error", err)
	}

	return &VPPFIPRouteStorage{
		db: db,
	}
}

func (s *VPPFIPRouteStorage) AddFIPRoute(fipRoute *model.VPPIPRoute) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if err := txn.Insert(VPPFIPRouteTableName, fipRoute); err != nil {
		return err
	}

	return nil
}

func (s *VPPFIPRouteStorage) DelFIPRoute(fipPrefix string) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	deleted, err := txn.DeleteAll(VPPFIPRouteTableName, "id", fipPrefix)
	if err != nil {
		return err
	}

	if deleted == 0 {
		return ErrNoVPPFIPFoundInStorage
	}

	return nil
}

func (s *VPPFIPRouteStorage) DelFIPRoutes() error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if _, err := txn.DeleteAll(VPPFIPRouteTableName, "id_prefix", ""); err != nil {
		return err
	}

	return nil
}

func (s *VPPFIPRouteStorage) GetFIPRoute(fipPrefix string) *model.VPPIPRoute {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPFIPRouteTableName, "id", fipPrefix)
	if err != nil {
		return nil
	}

	route, ok := raw.(*model.VPPIPRoute)
	if !ok {
		return nil
	}

	return route
}

func (s *VPPFIPRouteStorage) GetFIPRoutes() []*model.VPPIPRoute {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(VPPFIPRouteTableName, "id_prefix", "")
	if err != nil {
		return nil
	}

	if raws == nil {
		return nil
	}

	routes := make([]*model.VPPIPRoute, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		route, ok := r.(*model.VPPIPRoute)
		if ok {
			routes = append(routes, route)
		}
	}

	if len(routes) == 0 {
		return nil
	}

	return routes
}

func (s *VPPFIPRouteStorage) IsFIPPrefixExist(fipPrefix string) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPFIPRouteTableName, "id", fipPrefix)
	if err != nil {
		return false
	}

	_, ok := raw.(*model.VPPIPRoute)

	return ok
}

// IsFIPWithNHAndLabelExist checks if floating ip and nexthop with the same mpls label exists in imdb
func (s *VPPFIPRouteStorage) IsFIPWithNHAndLabelExist(fipPrefix, nextHop string, mplsLabel uint32) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPFIPRouteTableName, "id", fipPrefix)
	if err != nil {
		return false
	}

	route, ok := raw.(*model.VPPIPRoute)
	if !ok {
		return false
	}

	if len(route.FIPMPLSLabels) == 0 {
		return false
	}

	for i := range route.NextHops {
		if route.NextHops[i] == nextHop && route.FIPMPLSLabels[i] == mplsLabel {
			return true
		}
	}

	return false
}

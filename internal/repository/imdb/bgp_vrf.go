package imdb

import (
	"github.com/hashicorp/go-memdb"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var BGPVRFTableName = "bgp_vrf"

type BGPVRFStorage struct {
	db *memdb.MemDB
}

func NewBGPVRFStorage() *BGPVRFStorage {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			BGPVRFTableName: {
				Name: BGPVRFTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "ID"},
					},

					"name": {
						Name:    "name",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		logger.Fatal("failed to create bgp vrf storage", "error", err)
	}

	return &BGPVRFStorage{db: db}
}

func (s *BGPVRFStorage) AddVRF(vrf *model.BGPVRFTable) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if err := txn.Insert(BGPVRFTableName, vrf); err != nil {
		return err
	}

	return nil
}

func (s *BGPVRFStorage) DelVRFs() error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if _, err := txn.DeleteAll(BGPVRFTableName, "name_prefix", ""); err != nil {
		return err
	}

	return nil
}

func (s *BGPVRFStorage) GetVRF(vrfID uint32) *model.BGPVRFTable {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(BGPVRFTableName, "id", vrfID)
	if err != nil {
		return nil
	}

	vrf, ok := raw.(*model.BGPVRFTable)
	if !ok {
		return nil
	}

	return vrf
}

func (s *BGPVRFStorage) GetVRFs() []*model.BGPVRFTable {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(BGPVRFTableName, "name_prefix", "")
	if err != nil {
		return nil
	}

	if raws == nil {
		return nil
	}

	vrfs := make([]*model.BGPVRFTable, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		vrf, ok := r.(*model.BGPVRFTable)
		if ok {
			vrfs = append(vrfs, vrf)
		}
	}

	if len(vrfs) == 0 {
		return nil
	}

	return vrfs
}

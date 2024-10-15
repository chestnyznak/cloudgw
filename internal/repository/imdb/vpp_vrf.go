package imdb

import (
	"github.com/hashicorp/go-memdb"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var VPPVRFTableName = "vpp_vrf"

type VPPVRFStorage struct {
	db *memdb.MemDB
}

func NewVPPVRFStorage() *VPPVRFStorage {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			VPPVRFTableName: {
				Name: VPPVRFTableName,
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
		logger.Fatal("failed to create vpp vrf storage", "error", err)
	}

	return &VPPVRFStorage{
		db: db,
	}
}

func (s *VPPVRFStorage) AddVRF(vrf *model.VPPVRFTable) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if err := txn.Insert(VPPVRFTableName, vrf); err != nil {
		return err
	}

	return nil
}

func (s *VPPVRFStorage) DelVRFs() error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if _, err := txn.DeleteAll(VPPVRFTableName, "name_prefix", ""); err != nil {
		return err
	}

	return nil
}

func (s *VPPVRFStorage) GetVRF(vrfID uint32) *model.VPPVRFTable {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPVRFTableName, "id", vrfID)
	if err != nil {
		return nil
	}

	vrf, ok := raw.(*model.VPPVRFTable)
	if !ok {
		return nil
	}

	return vrf
}

func (s *VPPVRFStorage) GetVRFs() []*model.VPPVRFTable {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(VPPVRFTableName, "name_prefix", "")
	if err != nil {
		return nil
	}

	if raws == nil {
		return nil
	}

	vrfs := make([]*model.VPPVRFTable, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		vrf, ok := r.(*model.VPPVRFTable)
		if ok {
			vrfs = append(vrfs, vrf)
		}
	}

	if len(vrfs) == 0 {
		return nil
	}

	return vrfs
}

func (s *VPPVRFStorage) IncFIPServed(vrfID uint32) {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPVRFTableName, "id", vrfID)
	if err != nil {
		return
	}

	vrf, ok := raw.(*model.VPPVRFTable)
	if !ok {
		return
	}

	vrf.FIPServed++

	if err := txn.Insert(VPPVRFTableName, vrf); err != nil {
		return
	}
}

func (s *VPPVRFStorage) DecFIPServed(vrfID uint32) {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPVRFTableName, "id", vrfID)
	if err != nil {
		return
	}

	vrf, ok := raw.(*model.VPPVRFTable)
	if !ok {
		return
	}

	if vrf.FIPServed == 0 {
		return
	}

	vrf.FIPServed--

	if err := txn.Insert(VPPVRFTableName, vrf); err != nil {
		return
	}
}

func (s *VPPVRFStorage) GetFIPServed(vrfID uint32) uint32 {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPVRFTableName, "id", vrfID)
	if err != nil {
		return 0
	}

	vrf, ok := raw.(*model.VPPVRFTable)
	if !ok {
		return 0
	}

	return vrf.FIPServed
}

func (s *VPPVRFStorage) IsVRFExist(vrfID uint32) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPVRFTableName, "id", vrfID)
	if err != nil {
		return false
	}

	_, ok := raw.(*model.VPPVRFTable)

	return ok
}

func (s *VPPVRFStorage) CreateVRFIDToNextHopMap() (map[uint32]string, error) {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(VPPVRFTableName, "name_prefix", "")
	if err != nil {
		return nil, err
	}

	if raws == nil {
		return nil, ErrNoVPPVRFsFoundInStorage
	}

	result := make(map[uint32]string, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		vrf, ok := r.(*model.VPPVRFTable)
		if ok {
			result[vrf.ID] = vrf.NextHop
		}
	}

	if len(result) == 0 {
		return nil, ErrNoVPPVRFsFoundInStorage
	}

	return result, nil
}

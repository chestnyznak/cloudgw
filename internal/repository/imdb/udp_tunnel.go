package imdb

import (
	"github.com/hashicorp/go-memdb"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var VPPUDPTunnelTableName = "udp_tunnel"

type VPPUDPTunnelStorage struct {
	db *memdb.MemDB
}

func NewVPPUDPTunnelStorage() *VPPUDPTunnelStorage {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			VPPUDPTunnelTableName: {
				Name: VPPUDPTunnelTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "DstIP"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		logger.Fatal("failed to create vpp udp tunnel storage", "error", err)
	}

	return &VPPUDPTunnelStorage{
		db: db,
	}
}

func (s *VPPUDPTunnelStorage) AddUDPTunnel(tunnel *model.VPPUDPTunnel) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if err := txn.Insert(VPPUDPTunnelTableName, tunnel); err != nil {
		return err
	}

	return nil
}

func (s *VPPUDPTunnelStorage) DelUDPTunnel(dstIP string) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	deleted, err := txn.DeleteAll(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return err
	}

	if deleted == 0 {
		return ErrNoVPPUDPTunnelFoundInStorage
	}

	return nil
}

func (s *VPPUDPTunnelStorage) DelUDPTunnels() error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if _, err := txn.DeleteAll(VPPUDPTunnelTableName, "id_prefix", ""); err != nil {
		return err
	}

	return nil
}

func (s *VPPUDPTunnelStorage) GetUDPTunnel(dstIP string) *model.VPPUDPTunnel {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return nil
	}

	tunnel, ok := raw.(*model.VPPUDPTunnel)
	if !ok {
		return nil
	}

	return tunnel
}

func (s *VPPUDPTunnelStorage) GetUDPTunnels() []*model.VPPUDPTunnel {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(VPPUDPTunnelTableName, "id_prefix", "")
	if err != nil {
		return nil
	}

	if raws == nil {
		return nil
	}

	tunnels := make([]*model.VPPUDPTunnel, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		tunnel, ok := r.(*model.VPPUDPTunnel)
		if ok {
			tunnels = append(tunnels, tunnel)
		}
	}

	if len(tunnels) == 0 {
		return nil
	}

	return tunnels
}

func (s *VPPUDPTunnelStorage) IncFIPServed(dstIP string) {
	txn := s.db.Txn(true)

	defer txn.Commit()

	raw, err := txn.First(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return
	}

	tunnel, ok := raw.(*model.VPPUDPTunnel)
	if !ok {
		return
	}

	tunnel.FIPServed++

	if err := txn.Insert(VPPUDPTunnelTableName, tunnel); err != nil {
		return
	}
}

func (s *VPPUDPTunnelStorage) DecFIPServed(dstIP string) {
	txn := s.db.Txn(true)

	defer txn.Commit()

	raw, err := txn.First(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return
	}

	tunnel, ok := raw.(*model.VPPUDPTunnel)
	if !ok {
		return
	}

	if tunnel.FIPServed == 0 {
		return
	}

	tunnel.FIPServed--

	if err := txn.Insert(VPPUDPTunnelTableName, tunnel); err != nil {
		return
	}
}

func (s *VPPUDPTunnelStorage) GetFIPServed(dstIP string) uint32 {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return 0
	}

	tunnel, ok := raw.(*model.VPPUDPTunnel)
	if !ok {
		return 0
	}

	return tunnel.FIPServed
}

func (s *VPPUDPTunnelStorage) IsUDPTunnelExist(dstIP string) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(VPPUDPTunnelTableName, "id", dstIP)
	if err != nil {
		return false
	}

	_, ok := raw.(*model.VPPUDPTunnel)

	return ok
}

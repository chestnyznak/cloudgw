package imdb

import (
	"github.com/hashicorp/go-memdb"
	bgpapi "github.com/osrg/gobgp/v3/api"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var BGPPeerTableName = "bgp_peer"

type BGPPeerStorage struct {
	db *memdb.MemDB
}

func NewBGPPeerStorage() *BGPPeerStorage {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			BGPPeerTableName: {
				Name: BGPPeerTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "PeerAddress"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		logger.Fatal("failed to create bgp peer storage", "error", err)
	}

	return &BGPPeerStorage{db: db}
}

func (s *BGPPeerStorage) AddBGPPeer(bgpPeer *model.BGPPeer) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if err := txn.Insert(BGPPeerTableName, bgpPeer); err != nil {
		return err
	}

	return nil
}

func (s *BGPPeerStorage) DelBGPPeers() error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	if _, err := txn.DeleteAll(BGPPeerTableName, "id_prefix", ""); err != nil {
		return err
	}

	return nil
}

func (s *BGPPeerStorage) GetBGPPeer(peerIP string) *model.BGPPeer {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return nil
	}

	peer, ok := raw.(*model.BGPPeer)
	if !ok {
		return nil
	}

	return peer
}

func (s *BGPPeerStorage) GetBGPPeers() []*model.BGPPeer {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(BGPPeerTableName, "id_prefix", "")
	if err != nil {
		return nil
	}

	if raws == nil {
		return nil
	}

	peers := make([]*model.BGPPeer, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		peer, ok := r.(*model.BGPPeer)
		if ok {
			peers = append(peers, peer)
		}
	}

	if len(peers) == 0 {
		return nil
	}

	return peers
}

func (s *BGPPeerStorage) IsConfiguredBGPPeer(peerIP string) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return false
	}

	_, ok := raw.(*model.BGPPeer)

	return ok
}

func (s *BGPPeerStorage) CreateBGPPeerToTypeMap() (map[string]int, error) {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raws, err := txn.Get(BGPPeerTableName, "id_prefix", "")
	if err != nil {
		return nil, ErrNoBGPPeersFoundInStorage
	}

	if raws == nil {
		return nil, ErrNoBGPPeersFoundInStorage
	}

	peers := make([]*model.BGPPeer, 0)

	for r := raws.Next(); r != nil; r = raws.Next() {
		peer, ok := r.(*model.BGPPeer)
		if ok {
			peers = append(peers, peer)
		}
	}

	if len(peers) == 0 {
		return nil, ErrNoBGPPeersFoundInStorage
	}

	result := make(map[string]int, len(peers))

	for _, peer := range peers {
		result[peer.PeerAddress] = peer.PeerType
	}

	return result, nil
}

func (s *BGPPeerStorage) IsTF(peerIP string) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return false
	}

	peer, ok := raw.(*model.BGPPeer)
	if !ok {
		return false
	}

	return peer.PeerType == model.TF
}

func (s *BGPPeerStorage) IsPHYNET(peerIP string) bool {
	txn := s.db.Txn(false)

	defer txn.Abort()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return false
	}

	peer, ok := raw.(*model.BGPPeer)
	if !ok {
		return false
	}

	return peer.PeerType == model.PHYNET
}

func (s *BGPPeerStorage) UpdateBGPPeerState(peerIP string, prevState, currentState bgpapi.PeerState_SessionState) error {
	txn := s.db.Txn(true)

	defer txn.Commit()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return err
	}

	peer, ok := raw.(*model.BGPPeer)
	if !ok {
		return ErrNoBGPPeerFoundInStorage
	}

	peer.BGPPeerPrevState = prevState
	peer.BGPPeerState = currentState

	if err = txn.Insert(BGPPeerTableName, peer); err != nil {
		return err
	}

	return nil
}

func (s *BGPPeerStorage) UpdateBFDPeerState(peerIP string, isBFDEstablished bool) {
	txn := s.db.Txn(true)

	defer txn.Commit()

	raw, err := txn.First(BGPPeerTableName, "id", peerIP)
	if err != nil {
		return
	}

	peer, ok := raw.(*model.BGPPeer)
	if !ok {
		return
	}

	peer.BFDPeering.BFDPeerEstablished = isBFDEstablished

	if err := txn.Insert(BGPPeerTableName, peer); err != nil {
		return
	}
}

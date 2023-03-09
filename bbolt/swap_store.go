package bbolt

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/elementsproject/peerswap/swap"
	"go.etcd.io/bbolt"
)

var (
	swapBuckets          = []byte("swaps")
	requestedSwapsBucket = []byte("requested-swaps")

	ErrAlreadyExists = fmt.Errorf("swap already exist")
)

type BBoltSwapStore struct {
	db *bbolt.DB
}

func NewBBoltSwapStore(db *bbolt.DB) (*BBoltSwapStore, error) {
	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.CreateBucketIfNotExists(swapBuckets)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BBoltSwapStore{db: db}, nil
}

func (p *BBoltSwapStore) UpdateData(s *swap.SwapStateMachine) error {
	err := p.update(s)
	if err == swap.ErrDoesNotExist {
		err = nil
		err = p.create(s)
	}
	if err != nil {
		return err
	}
	return nil

}

func (p *BBoltSwapStore) GetData(id string) (*swap.SwapStateMachine, error) {
	s, err := p.getById(id)
	if err == swap.ErrDoesNotExist {
		return nil, swap.ErrDataNotAvailable
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (p *BBoltSwapStore) create(swap *swap.SwapStateMachine) error {
	exists, err := p.idExists(swap.SwapId.String())
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyExists
	}

	tx, err := p.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(swapBuckets)
	if b == nil {
		return fmt.Errorf("bucket nil")
	}

	jData, err := json.Marshal(swap)
	if err != nil {
		return err
	}

	if err := b.Put(h2b(swap.SwapId.String()), jData); err != nil {
		return err
	}

	return tx.Commit()
}

func (p *BBoltSwapStore) update(s *swap.SwapStateMachine) error {
	exists, err := p.idExists(s.SwapId.String())
	if err != nil {
		return err
	}
	if !exists {
		return swap.ErrDoesNotExist
	}
	tx, err := p.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(swapBuckets)
	if b == nil {
		return fmt.Errorf("bucket nil")
	}
	jData, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if err := b.Put(h2b(s.SwapId.String()), jData); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *BBoltSwapStore) getById(s string) (*swap.SwapStateMachine, error) {
	tx, err := p.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	b := tx.Bucket(swapBuckets)
	if b == nil {
		return nil, fmt.Errorf("bucket nil")
	}

	jData := b.Get(h2b(s))
	if jData == nil {
		return nil, swap.ErrDoesNotExist
	}

	sw := &swap.SwapStateMachine{}
	if err := json.Unmarshal(jData, sw); err != nil {
		return nil, err
	}

	return sw, nil
}

func (p *BBoltSwapStore) ListAll() ([]*swap.SwapStateMachine, error) {
	tx, err := p.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	b := tx.Bucket(swapBuckets)
	if b == nil {
		return nil, fmt.Errorf("bucket nil")
	}
	var swaps []*swap.SwapStateMachine
	err = b.ForEach(func(k, v []byte) error {

		s := &swap.SwapStateMachine{}
		if err := json.Unmarshal(v, s); err != nil {
			return err
		}
		swaps = append(swaps, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return swaps, nil
}

func (p *BBoltSwapStore) ListAllByPeer(peer string) ([]*swap.SwapStateMachine, error) {
	tx, err := p.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	b := tx.Bucket(swapBuckets)
	if b == nil {
		return nil, fmt.Errorf("bucket nil")
	}

	var swaps []*swap.SwapStateMachine
	err = b.ForEach(func(k, v []byte) error {
		s := &swap.SwapStateMachine{}
		if err := json.Unmarshal(v, s); err != nil {
			return err
		}
		if s.Data.PeerNodeId == peer {
			swaps = append(swaps, s)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return swaps, nil
}

func (p *BBoltSwapStore) idExists(id string) (bool, error) {
	_, err := p.getById(id)
	if err != nil {
		if err == swap.ErrDoesNotExist {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func h2b(str string) []byte {
	buf, _ := hex.DecodeString(str)
	return buf
}

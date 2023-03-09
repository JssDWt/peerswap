package bbolt

import (
	"encoding/json"

	"github.com/elementsproject/peerswap/swap"
	"go.etcd.io/bbolt"
)

type BBoltRequestedSwapsStore struct {
	db *bbolt.DB
}

func NewBBoltRequestedSwapsStore(db *bbolt.DB) (*BBoltRequestedSwapsStore, error) {
	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.CreateBucketIfNotExists(requestedSwapsBucket)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BBoltRequestedSwapsStore{db: db}, nil
}

func (s *BBoltRequestedSwapsStore) Add(id string, reqswap swap.RequestedSwap) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(requestedSwapsBucket)
		k := b.Get([]byte(id))

		var reqswaps []swap.RequestedSwap
		if k == nil {
			reqswaps = []swap.RequestedSwap{reqswap}
		} else {
			err := json.Unmarshal(k, &reqswaps)
			if err != nil {
				return err
			}

			reqswaps = append(reqswaps, reqswap)
		}

		buf, err := json.Marshal(reqswaps)
		if err != nil {
			return err
		}

		return b.Put([]byte(id), buf)
	})
}

func (s *BBoltRequestedSwapsStore) GetAll() (map[string][]swap.RequestedSwap, error) {
	reqswaps := map[string][]swap.RequestedSwap{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(requestedSwapsBucket)
		return b.ForEach(func(k, v []byte) error {
			id := string(k)
			var reqswap []swap.RequestedSwap
			json.Unmarshal(v, &reqswap)
			reqswaps[id] = reqswap
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return reqswaps, nil
}

func (s *BBoltRequestedSwapsStore) Get(id string) ([]swap.RequestedSwap, error) {
	var reqswaps []swap.RequestedSwap
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(requestedSwapsBucket)
		k := b.Get([]byte(id))
		return json.Unmarshal(k, &reqswaps)
	})
	if err != nil {
		return nil, err
	}

	return reqswaps, nil
}

package bbolt

import (
	"encoding/json"
	"time"

	"github.com/elementsproject/peerswap/poll"

	"go.etcd.io/bbolt"
)

var POLL_BUCKET = []byte("poll-list")

type BBoltPollStore struct {
	db *bbolt.DB
}

func NewBBoltPollStore(db *bbolt.DB) (*BBoltPollStore, error) {
	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.CreateBucketIfNotExists(POLL_BUCKET)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BBoltPollStore{db: db}, nil
}

func (s *BBoltPollStore) Update(peerId string, info poll.PollInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		infoBytes, err := json.Marshal(info)
		if err != nil {
			return err
		}

		b := tx.Bucket(POLL_BUCKET)
		return b.Put([]byte(peerId), infoBytes)
	})
}

func (s *BBoltPollStore) GetAll() (map[string]poll.PollInfo, error) {
	pollinfos := map[string]poll.PollInfo{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(POLL_BUCKET)
		return b.ForEach(func(k, v []byte) error {
			peerId := string(k)
			var info poll.PollInfo
			err := json.Unmarshal(v, &info)
			if err != nil {
				return err
			}
			pollinfos[peerId] = info
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return pollinfos, nil
}

func (s *BBoltPollStore) RemoveUnseen(olderThan time.Duration) error {
	now := time.Now()
	return s.db.Update(func(t *bbolt.Tx) error {
		b := t.Bucket(POLL_BUCKET)
		return b.ForEach(func(k, v []byte) error {
			var info poll.PollInfo
			err := json.Unmarshal(v, &info)
			if err != nil {
				return err
			}
			if now.Sub(info.LastSeen) > olderThan {
				b.Delete(k)
			}
			return nil
		})
	})
}

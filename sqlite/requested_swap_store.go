package sqlite

import (
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/elementsproject/peerswap/swap"
	_ "modernc.org/sqlite" // Register relevant drivers.
)

type SqliteRequestedSwapStore struct {
	db *sql.DB
}

func NewSqliteRequestedSwapStore(db *sql.DB) (*SqliteRequestedSwapStore, error) {
	return &SqliteRequestedSwapStore{db: db}, nil
}

func (s *SqliteRequestedSwapStore) Add(id string, reqswap swap.RequestedSwap) error {
	peerid, err := hex.DecodeString(id)
	if err != nil {
		return fmt.Errorf("failed to decode peer id '%s': %w", id, err)
	}

	_, err = s.db.Exec(
		`INSERT INTO requested_swaps (peer_node_id, asset, amount_sat, type, rejection_reason)
		VALUES (?,?,?,?,?)`, peerid, reqswap.Asset, int(reqswap.Type), int64(reqswap.AmountSat), reqswap.RejectionReason)
	if err != nil {
		return fmt.Errorf("failed to insert requested swap: %w", err)
	}

	return nil
}

func (s *SqliteRequestedSwapStore) Get(id string) ([]swap.RequestedSwap, error) {
	peerid, err := hex.DecodeString(id)
	if err != nil {
		return nil, fmt.Errorf("failed to decode peer id '%s': %w", id, err)
	}

	rows, err := s.db.Query(
		`SELECT asset, amount_sat, type, rejection_reason
		 FROM requested_swaps
		 WHERE peer_node_id = ?`, peerid)
	if err != nil {
		return nil, err
	}

	var result []swap.RequestedSwap
	for rows.Next() {
		var asset string
		var amount_sat int64
		var t int
		var rejection_reason string
		err = rows.Scan(&asset, &amount_sat, &t, &rejection_reason)
		if err != nil {
			return nil, err
		}

		result = append(result, swap.RequestedSwap{
			Asset:           asset,
			AmountSat:       uint64(amount_sat),
			Type:            swap.SwapType(t),
			RejectionReason: rejection_reason,
		})
	}

	return result, nil
}

func (s *SqliteRequestedSwapStore) GetAll() (map[string][]swap.RequestedSwap, error) {
	rows, err := s.db.Query(
		`SELECT peer_node_id, asset, amount_sat, type, rejection_reason
		 FROM requested_swaps
		 ORDER BY peer_node_id, asset, type, amount_sat`)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]swap.RequestedSwap)
	var lastPeerId string
	for rows.Next() {
		var peerid []byte
		var asset string
		var amount_sat int64
		var t int
		var rejection_reason string
		err = rows.Scan(&peerid, &asset, &amount_sat, &t, &rejection_reason)
		if err != nil {
			return nil, err
		}

		r := swap.RequestedSwap{
			Asset:           asset,
			AmountSat:       uint64(amount_sat),
			Type:            swap.SwapType(t),
			RejectionReason: rejection_reason,
		}

		currentPeerId := hex.EncodeToString(peerid)
		if lastPeerId != "" && currentPeerId == lastPeerId {
			result[currentPeerId] = append(result[currentPeerId], r)
		} else {
			result[currentPeerId] = []swap.RequestedSwap{r}
		}

		lastPeerId = currentPeerId
	}

	return result, nil
}

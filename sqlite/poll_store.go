package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/elementsproject/peerswap/poll"
	_ "modernc.org/sqlite" // Register relevant drivers.
)

type SqlitePollStore struct {
	db *sql.DB
}

func NewSqlitePollStore(db *sql.DB) (*SqlitePollStore, error) {
	return &SqlitePollStore{db: db}, nil
}

func (s *SqlitePollStore) Update(peerId string, info poll.PollInfo) error {
	tx, err := s.db.BeginTx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	peerIdBytes, err := hex.DecodeString(peerId)
	if err != nil {
		return fmt.Errorf("failed to decode peer id: %w", err)
	}

	_, err = tx.Exec("DELETE FROM poll_info_assets WHERE peer_node_id = ?", peerIdBytes)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete poll_info assets for peer '%s': %w", peerId, err)
	}
	_, err = tx.Exec("DELETE FROM poll_info WHERE peer_node_id = ?", peerIdBytes)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete poll_info for peer '%s': %w", peerId, err)
	}
	_, err = tx.Exec(
		`INSERT INTO poll_info (peer_node_id, protocol_version, peer_allowed, last_seen)
		 VALUES (?,?,?,?)`,
		peerIdBytes, info.ProtocolVersion, info.PeerAllowed, info.LastSeen.Unix())
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert poll_info for peer '%s': %w", peerId, err)
	}

	if len(info.Assets) > 0 {
		assetSql := `INSERT INTO poll_info_assets (peer_node_id, asset) VALUES `
		vals := []interface{}{}
		for _, a := range info.Assets {
			assetSql += "(?,?),"
			vals = append(vals, peerIdBytes, a)
		}
		assetSql = strings.TrimSuffix(assetSql, ",")
		_, err = tx.Exec(assetSql, vals...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert poll_info assets for peer '%s': %w", peerId, err)
		}
	}

	return tx.Commit()
}

func (s *SqlitePollStore) GetAll() (map[string]poll.PollInfo, error) {
	rows, err := s.db.Query(
		`SELECT i.peer_node_id
		,       i.protocol_version
		,       i.peer_allowed
		,       i.last_seen
		,       a.asset
		FROM poll_info i
		LEFT JOIN poll_info_assets a ON i.peer_node_id = a.peer_node_id
		ORDER BY i.peer_node_id, a.asset`)
	if err != nil {
		return nil, err
	}

	result := make(map[string]poll.PollInfo)
	var lastPeerId []byte
	var lastInfo *poll.PollInfo
	for rows.Next() {
		var peerId []byte
		var protocol_version uint64
		var peer_allowed bool
		var last_seen int64
		var asset string
		err = rows.Scan(&peerId, &protocol_version, &peer_allowed, &last_seen, &asset)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if lastPeerId != nil && bytes.Equal(lastPeerId, peerId) {
			lastInfo.Assets = append(lastInfo.Assets, asset)
			continue
		}

		lastPeerId = peerId
		lastInfo = &poll.PollInfo{
			ProtocolVersion: protocol_version,
			Assets:          []string{asset},
			PeerAllowed:     peer_allowed,
			LastSeen:        time.Unix(last_seen, 0),
		}
		result[hex.EncodeToString(peerId)] = *lastInfo
	}

	return result, nil
}

func (s *SqlitePollStore) RemoveUnseen(olderThan time.Duration) error {
	before := time.Now().Add(-olderThan).Unix()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`DELETE FROM poll_info_assets a WHERE a.peer_node_id IN (
			SELECT i.peer_node_id
			FROM poll_info i
			WHERE i.last_seen < ?
		)`, before)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM poll_info WHERE last_seen < ?`)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

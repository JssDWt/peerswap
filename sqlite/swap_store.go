package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/elementsproject/peerswap/swap"
	_ "modernc.org/sqlite" // Register relevant drivers.
)

type SqliteSwapStore struct {
	db *sql.DB
}

func NewSqliteSwapStore(db *sql.DB) (*SqliteSwapStore, error) {
	return &SqliteSwapStore{db: db}, nil
}

func (p *SqliteSwapStore) UpdateData(s *swap.SwapStateMachine) error {
	// TODO(JssDWt): Break calls to this function down to smaller pieces so only
	// relevant parts are updated.
	tx, err := p.db.BeginTx(context.TODO(), &sql.TxOptions{
		Isolation: sql.LevelSnapshot,
	})
	if err != nil {
		return err
	}

	peerid, err := hex.DecodeString(s.Data.PeerNodeId)
	if err != nil {
		return fmt.Errorf("failed to decode peer node id '%s': %w", s.Data.PeerNodeId, err)
	}

	initiatorid, err := hex.DecodeString(s.Data.InitiatorNodeId)
	if err != nil {
		return fmt.Errorf("failed to decode initiator node id '%s': %w", s.Data.InitiatorNodeId, err)
	}

	feePreimage, err := hex.DecodeString(s.Data.FeePreimage)
	if err != nil {
		return fmt.Errorf("failed to decode fee preimage '%s': %w", s.Data.FeePreimage, err)
	}

	openingtx, err := hex.DecodeString(s.Data.OpeningTxHex)
	if err != nil {
		return fmt.Errorf("failed to decode opening tx hex '%s': %w", s.Data.OpeningTxHex, err)
	}

	claimtxid, err := hex.DecodeString(s.Data.ClaimTxId)
	if err != nil {
		return fmt.Errorf("failed to decode claim tx id '%s': %w", s.Data.ClaimTxId, err)
	}

	claimpreimage, err := hex.DecodeString(s.Data.ClaimPreimage)
	if err != nil {
		return fmt.Errorf("failed to decode claim preimage '%s': %w", s.Data.ClaimPreimage, err)
	}

	claimPaymentHash, err := hex.DecodeString(s.Data.ClaimPaymentHash)
	if err != nil {
		return fmt.Errorf("failed to decode claim payment hash '%s': %w", s.Data.ClaimPaymentHash, err)
	}

	blindingkey, err := hex.DecodeString(s.Data.BlindingKeyHex)
	if err != nil {
		return fmt.Errorf("failed to decode blinding key '%s': %w", s.Data.BlindingKeyHex, err)
	}

	_, err = tx.Exec(`INSERT INTO swaps (
		swap_id
	,	type
	,	role
	,	previous_state
	,	current_state
	,	peer_node_id
	,	initiator_node_id
	,	created_at
	,	private_key
	,	fee_preimage
	,	opening_tx_fee
	,	opening_tx
	,	starting_block_height
	,	claim_tx_id
	,	claim_payment_hash
	,	claim_preimage
	,	blinding_key
	,	next_message
	,	next_message_type
	,	last_err
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(swap_id) DO UPDATE SET
		type=excluded.type
	,	role=excluded.role
	,	previous_state=excluded.previous_state
	,	current_state=excluded.current_state
	,	peer_node_id=excluded.peer_node_id
	,	initiator_node_id=excluded.initiator_node_id
	,	created_at=excluded.created_at
	,	private_key=excluded.private_key
	,	fee_preimage=excluded.fee_preimage
	,	opening_tx_fee=excluded.opening_tx_fee
	,	opening_tx=excluded.opening_tx
	,	starting_block_height=excluded.starting_block_height
	,	claim_tx_id=excluded.claim_tx_id
	,	claim_payment_hash=excluded.claim_payment_hash
	,	claim_preimage=excluded.claim_preimage
	,	blinding_key=excluded.blinding_key
	,	next_message=excluded.next_message
	,	next_message_type=excluded.next_message_type
	,	last_err=excluded.last_err`,
		s.SwapId[:],
		int(s.Type),
		int(s.Role),
		s.Previous,
		s.Current,
		peerid,
		initiatorid,
		s.Data.CreatedAt,
		s.Data.PrivkeyBytes,
		feePreimage,
		int64(s.Data.OpeningTxFee),
		openingtx,
		s.Data.StartingBlockHeight,
		claimtxid,
		claimPaymentHash,
		claimpreimage,
		blindingkey,
		s.Data.NextMessage,
		s.Data.NextMessageType,
		s.Data.LastErrString,
	)

	return err
}

func (p *SqliteSwapStore) GetData(id string) (*swap.SwapStateMachine, error) {
	swapid, err := hex.DecodeString(id)
	if err != nil {
		return nil, fmt.Errorf("failed to decode swap id '%s': %w", id, err)
	}

	row := p.db.QueryRow(`SELECT type
	,	role
	,	previous_state
	,	current_state
	,	peer_node_id
	,	initiator_node_id
	,	created_at
	,	private_key
	,	fee_preimage
	,	opening_tx_fee
	,	opening_tx
	,	starting_block_height
	,	claim_tx_id
	,	claim_payment_hash
	,	claim_preimage
	,	blinding_key
	,	next_message
	,	next_message_type
	,	last_err
	FROM swaps
	WHERE swap_id = ?`, swapid)
	var t int
	var role int
	var previous string
	var current string
	var peer_node_id []byte
	var initiator_node_id []byte
	var created_at int64
	var private_key []byte
	var fee_preimage []byte
	var opening_tx_fee int64
	var opening_tx []byte
	var starting_block_height uint32
	var claim_tx_id []byte
	var claim_payment_hash []byte
	var claim_preimage []byte
	var blinding_key []byte
	var next_message []byte
	var next_message_type int
	var last_err string
	err = row.Scan(
		&t,
		&role,
		&previous,
		&current,
		&peer_node_id,
		&initiator_node_id,
		&created_at,
		&private_key,
		&fee_preimage,
		&opening_tx_fee,
		&opening_tx,
		&starting_block_height,
		&claim_tx_id,
		&claim_payment_hash,
		&claim_preimage,
		&blinding_key,
		&next_message,
		&next_message_type,
		&last_err,
	)
	if err == sql.ErrNoRows {
		return nil, swap.ErrDataNotAvailable
	}
	if err != nil {
		return nil, err
	}

	resid := &swap.SwapId{}
	err = resid.FromString(id)
	if err != nil {
		return nil, err
	}

	return &swap.SwapStateMachine{
		SwapId:   resid,
		Type:     swap.SwapType(t),
		Role:     swap.SwapRole(role),
		Previous: swap.StateType(previous),
		Current:  swap.StateType(current),
		Data: &swap.SwapData{
			PeerNodeId:          hex.EncodeToString(peer_node_id),
			InitiatorNodeId:     hex.EncodeToString(initiator_node_id),
			CreatedAt:           created_at,
			Role:                swap.SwapRole(role),
			FSMState:            swap.StateType(current),
			PrivkeyBytes:        private_key,
			FeePreimage:         hex.EncodeToString(fee_preimage),
			OpeningTxFee:        uint64(opening_tx_fee),
			OpeningTxHex:        hex.EncodeToString(opening_tx),
			StartingBlockHeight: starting_block_height,
			ClaimTxId:           hex.EncodeToString(claim_tx_id),
			ClaimPaymentHash:    hex.EncodeToString(claim_payment_hash),
			ClaimPreimage:       hex.EncodeToString(claim_preimage),
			BlindingKeyHex:      hex.EncodeToString(blinding_key),
			NextMessage:         next_message,
			NextMessageType:     next_message_type,
			LastErrString:       last_err,
		},
	}, nil
}

func (p *SqliteSwapStore) ListAll() ([]*swap.SwapStateMachine, error) {
	rows, err := p.db.Query(`SELECT swap_id
	,   type
	,	role
	,	previous_state
	,	current_state
	,	peer_node_id
	,	initiator_node_id
	,	created_at
	,	private_key
	,	fee_preimage
	,	opening_tx_fee
	,	opening_tx
	,	starting_block_height
	,	claim_tx_id
	,	claim_payment_hash
	,	claim_preimage
	,	blinding_key
	,	next_message
	,	next_message_type
	,	last_err
	FROM swaps`)
	if err != nil {
		return nil, err
	}

	var result []*swap.SwapStateMachine
	for rows.Next() {
		var swap_id []byte
		var t int
		var role int
		var previous string
		var current string
		var peer_node_id []byte
		var initiator_node_id []byte
		var created_at int64
		var private_key []byte
		var fee_preimage []byte
		var opening_tx_fee int64
		var opening_tx []byte
		var starting_block_height uint32
		var claim_tx_id []byte
		var claim_payment_hash []byte
		var claim_preimage []byte
		var blinding_key []byte
		var next_message []byte
		var next_message_type int
		var last_err string
		err = rows.Scan(
			&swap_id,
			&t,
			&role,
			&previous,
			&current,
			&peer_node_id,
			&initiator_node_id,
			&created_at,
			&private_key,
			&fee_preimage,
			&opening_tx_fee,
			&opening_tx,
			&starting_block_height,
			&claim_tx_id,
			&claim_payment_hash,
			&claim_preimage,
			&blinding_key,
			&next_message,
			&next_message_type,
			&last_err,
		)

		if err != nil {
			return nil, err
		}

		resid := swap.SwapId{}
		copy(resid[:], swap_id[:])

		result = append(result, &swap.SwapStateMachine{
			SwapId:   &resid,
			Type:     swap.SwapType(t),
			Role:     swap.SwapRole(role),
			Previous: swap.StateType(previous),
			Current:  swap.StateType(current),
			Data: &swap.SwapData{
				PeerNodeId:          hex.EncodeToString(peer_node_id),
				InitiatorNodeId:     hex.EncodeToString(initiator_node_id),
				CreatedAt:           created_at,
				Role:                swap.SwapRole(role),
				FSMState:            swap.StateType(current),
				PrivkeyBytes:        private_key,
				FeePreimage:         hex.EncodeToString(fee_preimage),
				OpeningTxFee:        uint64(opening_tx_fee),
				OpeningTxHex:        hex.EncodeToString(opening_tx),
				StartingBlockHeight: starting_block_height,
				ClaimTxId:           hex.EncodeToString(claim_tx_id),
				ClaimPaymentHash:    hex.EncodeToString(claim_payment_hash),
				ClaimPreimage:       hex.EncodeToString(claim_preimage),
				BlindingKeyHex:      hex.EncodeToString(blinding_key),
				NextMessage:         next_message,
				NextMessageType:     next_message_type,
				LastErrString:       last_err,
			},
		})
	}

	return result, nil
}

func (p *SqliteSwapStore) ListAllByPeer(peer string) ([]*swap.SwapStateMachine, error) {
	peerid, err := hex.DecodeString(peer)
	if err != nil {
		return nil, fmt.Errorf("failed to decode peer node id '%s': %w", peer, err)
	}

	rows, err := p.db.Query(`SELECT swap_id
	,   type
	,	role
	,	previous_state
	,	current_state
	,	peer_node_id
	,	initiator_node_id
	,	created_at
	,	private_key
	,	fee_preimage
	,	opening_tx_fee
	,	opening_tx
	,	starting_block_height
	,	claim_tx_id
	,	claim_payment_hash
	,	claim_preimage
	,	blinding_key
	,	next_message
	,	next_message_type
	,	last_err
	FROM swaps
	WHERE peer_node_id = ?`,
		peerid)
	if err != nil {
		return nil, err
	}

	var result []*swap.SwapStateMachine
	for rows.Next() {
		var swap_id []byte
		var t int
		var role int
		var previous string
		var current string
		var peer_node_id []byte
		var initiator_node_id []byte
		var created_at int64
		var private_key []byte
		var fee_preimage []byte
		var opening_tx_fee int64
		var opening_tx []byte
		var starting_block_height uint32
		var claim_tx_id []byte
		var claim_payment_hash []byte
		var claim_preimage []byte
		var blinding_key []byte
		var next_message []byte
		var next_message_type int
		var last_err string
		err = rows.Scan(
			&swap_id,
			&t,
			&role,
			&previous,
			&current,
			&peer_node_id,
			&initiator_node_id,
			&created_at,
			&private_key,
			&fee_preimage,
			&opening_tx_fee,
			&opening_tx,
			&starting_block_height,
			&claim_tx_id,
			&claim_payment_hash,
			&claim_preimage,
			&blinding_key,
			&next_message,
			&next_message_type,
			&last_err,
		)

		if err != nil {
			return nil, err
		}

		resid := swap.SwapId{}
		copy(resid[:], swap_id[:])

		result = append(result, &swap.SwapStateMachine{
			SwapId:   &resid,
			Type:     swap.SwapType(t),
			Role:     swap.SwapRole(role),
			Previous: swap.StateType(previous),
			Current:  swap.StateType(current),
			Data: &swap.SwapData{
				PeerNodeId:          hex.EncodeToString(peer_node_id),
				InitiatorNodeId:     hex.EncodeToString(initiator_node_id),
				CreatedAt:           created_at,
				Role:                swap.SwapRole(role),
				FSMState:            swap.StateType(current),
				PrivkeyBytes:        private_key,
				FeePreimage:         hex.EncodeToString(fee_preimage),
				OpeningTxFee:        uint64(opening_tx_fee),
				OpeningTxHex:        hex.EncodeToString(opening_tx),
				StartingBlockHeight: starting_block_height,
				ClaimTxId:           hex.EncodeToString(claim_tx_id),
				ClaimPaymentHash:    hex.EncodeToString(claim_payment_hash),
				ClaimPreimage:       hex.EncodeToString(claim_preimage),
				BlindingKeyHex:      hex.EncodeToString(blinding_key),
				NextMessage:         next_message,
				NextMessageType:     next_message_type,
				LastErrString:       last_err,
			},
		})
	}

	return result, nil
}

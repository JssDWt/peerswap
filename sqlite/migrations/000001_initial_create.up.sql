-- TODO: nullability / normalization
CREATE TABLE swaps (
    swap_id BINARY(32)
,   type INTEGER
,   role INTEGER
,   previous_state NVARCHAR
,   current_state NVARCHAR

,   peer_node_id BINARY(33)
,   initiator_node_id BINARY(33)
,   created_at INTEGER
,   private_key BINARY(32)
,   fee_preimage BINARY(32)
,   opening_tx_fee INTEGER
,   opening_tx BINARY
,   starting_block_height INTEGER
,   claim_tx_id BINARY(32)
,   claim_payment_hash BINARY(32)
,   claim_preimage BINARY(32)
,   blinding_key BINARY(32)
,   next_message BINARY
,   next_message_type INTEGER
,   last_err NVARCHAR
);

CREATE TABLE swap_in_request_messages (
    swap_id BINARY(32)
,   protocol_version INTEGER
,   network NVARCHAR
,   asset NVARCHAR
,   scid VARCHAR
,   amount INTEGER
,   pubkey BINARY(33)
);

CREATE TABLE swap_in_agreement_messages (
    swap_id BINARY(32)
,   protocol_version INTEGER
,   pubkey BINARY(33)
,   premium INTEGER
);

CREATE TABLE swap_out_request_messages (
    swap_id BINARY(32)
,   protocol_version INTEGER
,   network NVARCHAR
,   asset NVARCHAR
,   scid VARCHAR
,   amount INTEGER
,   pubkey BINARY(33)
);

CREATE TABLE swap_out_agreement_messages (
    swap_id BINARY(32)
,   protocol_version INTEGER
,   pubkey BINARY(33)
,   payreq NVARCHAR
);

CREATE TABLE opening_tx_broadcasted_messages (
    swap_id BINARY(32)
,   payreq NVARCHAR
,   txid BINARY(32)
,   script_out INTEGER
,   blinding_key BINARY(32)
);

CREATE TABLE coop_close_messages (
    swap_id BINARY(32)
,   message NVARCHAR
,   privkey BINARY(32)
);

CREATE TABLE cancel_messages (
    swap_id BINARY(32)
,   message NVARCHAR
);

CREATE TABLE requested_swaps (
    peer_node_id BINARY(33)
,   asset NVARCHAR
,   amount_sat INTEGER
,   type INTEGER
,   rejection_reason NVARCHAR
);

CREATE TABLE poll_info (
    peer_node_id BINARY(33)
,   protocol_version INTEGER
,   peer_allowed BOOLEAN
,   last_seen INTEGER
);

CREATE TABLE poll_info_assets (
    peer_node_id BINARY(33)
,   asset NVARCHAR
);

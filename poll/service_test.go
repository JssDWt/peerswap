package poll

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elementsproject/peerswap/messages"
	"github.com/stretchr/testify/assert"
)

type MessengerMock struct {
	peersReceived   []string
	msgReceived     [][]byte
	msgTypeReceived []int
	called          uint
}

func (m *MessengerMock) SendMessage(peerId string, message []byte, messageType int) error {
	m.called++
	m.peersReceived = append(m.peersReceived, peerId)
	m.msgReceived = append(m.msgReceived, message)
	m.msgTypeReceived = append(m.msgTypeReceived, messageType)
	return nil
}

func (m *MessengerMock) AddMessageHandler(func(peerId string, msgType string, payload []byte) error) {
}

type PeerGetterMock struct {
	peers  []string
	called int
}

func (m *PeerGetterMock) GetPeers() []string {
	m.called++
	return m.peers
}

type PolicyMock struct {
	allowList []bool
	called    uint
}

func (m *PolicyMock) IsPeerAllowed(peerId string) bool {
	m.called++
	return m.allowList[m.called-1]
}

type PollStoreMock struct {
	infos map[string]PollInfo
}

func NewPollStoreMock() *PollStoreMock {
	return &PollStoreMock{
		infos: make(map[string]PollInfo),
	}
}

func (m *PollStoreMock) Update(peerId string, info PollInfo) error {
	m.infos[peerId] = info
	return nil
}

func (m *PollStoreMock) GetAll() (map[string]PollInfo, error) {
	return m.infos, nil
}

func (m *PollStoreMock) RemoveUnseen(olderThan time.Duration) error {
	now := time.Now()
	for peerid, info := range m.infos {
		if now.Sub(info.LastSeen) > olderThan {
			delete(m.infos, peerid)
		}
	}

	return nil
}

func TestSendMessage(t *testing.T) {
	store := NewPollStoreMock()
	messenger := &MessengerMock{}
	policy := &PolicyMock{allowList: []bool{true, false}}
	peerGetter := &PeerGetterMock{
		peers: []string{"peer1", "peer2"},
	}
	assets := []string{"asset1", "asset2", "asset3"}
	ps := NewService(500*time.Millisecond, 1*time.Second, store, messenger, policy, peerGetter, assets)
	for _, peer := range peerGetter.peers {
		ps.Poll(peer)
	}

	assert.Len(t, messenger.peersReceived, len(peerGetter.peers))
	assert.ElementsMatch(t, messenger.peersReceived, peerGetter.peers)

	var msgs []PollMessage
	for _, msgByte := range messenger.msgReceived {
		var msg PollMessage
		json.Unmarshal(msgByte, &msg)
		msgs = append(msgs, msg)
	}

	for i, isAllowed := range policy.allowList {
		assert.Equal(t, isAllowed, msgs[i].PeerAllowed)
		assert.Equal(t, msgs[i].Assets, assets)
	}
}

func TestRecievePollAndPollRequest(t *testing.T) {
	store := NewPollStoreMock()
	messenger := &MessengerMock{}
	policy := &PolicyMock{allowList: []bool{true, false}}
	peerGetter := &PeerGetterMock{
		peers: []string{"peer1", "peer2"},
	}
	assets := []string{"asset1", "asset2", "asset3"}
	ps := NewService(500*time.Millisecond, 1*time.Second, store, messenger, policy, peerGetter, assets)

	pmt := messages.MessageTypeToHexString(messages.MESSAGETYPE_POLL)
	rpmt := messages.MessageTypeToHexString(messages.MESSAGETYPE_REQUEST_POLL)

	// Handle poll message
	pmp, err := json.Marshal(PollMessage{
		Version:     0,
		Assets:      []string{},
		PeerAllowed: false,
	})
	if err != nil {
		t.Fatalf("could not marshal poll msg: %v", err)
	}
	ps.MessageHandler("peer", pmt, pmp)

	polls, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll(): %v", err)
	}
	assert.Len(t, polls, 1)
	assert.Len(t, messenger.peersReceived, 0)

	// Handle poll request message
	rpmp, err := json.Marshal(PollMessage{
		Version:     0,
		Assets:      []string{},
		PeerAllowed: false,
	})
	if err != nil {
		t.Fatalf("could not marshal poll msg: %v", err)
	}
	ps.MessageHandler("request-peer", rpmt, rpmp)

	polls, err = store.GetAll()
	if err != nil {
		t.Fatalf("GetAll(): %v", err)
	}
	assert.Len(t, polls, 2)
	assert.Len(t, messenger.peersReceived, 1)

	assert.ElementsMatch(t, messenger.peersReceived, []string{"request-peer"})
}

func TestRemoveUnseen(t *testing.T) {
	store := NewPollStoreMock()
	now := time.Now()
	duration := 10 * time.Millisecond

	err := store.Update("peer", PollInfo{
		Assets:      []string{"lbtc", "btc"},
		PeerAllowed: false,
		LastSeen:    now,
	})
	if err != nil {
		t.Fatalf("could not create store: %v", err)
	}
	err = store.Update("peer1", PollInfo{
		Assets:      []string{"lbtc", "btc"},
		PeerAllowed: false,
		LastSeen:    now.Add(duration),
	})
	if err != nil {
		t.Fatalf("could not create store: %v", err)
	}

	time.Sleep(duration)
	err = store.RemoveUnseen(duration)
	if err != nil {
		t.Fatalf("could not create store: %v", err)
	}
	m, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll(): %v", err)
	}

	assert.Len(t, m, 1)
}

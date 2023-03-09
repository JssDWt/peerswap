package swap

import "fmt"

var (
	ErrDoesNotExist = fmt.Errorf("does not exist")
)

type RequestedSwapsStore interface {
	Add(id string, reqswap RequestedSwap) error
	Get(id string) ([]RequestedSwap, error)
	GetAll() (map[string][]RequestedSwap, error)
}

type RequestedSwap struct {
	Asset           string   `json:"asset"`
	AmountSat       uint64   `json:"amount_sat"`
	Type            SwapType `json:"swap_type"`
	RejectionReason string   `json:"rejection_reason"`
}

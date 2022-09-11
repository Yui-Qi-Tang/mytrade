package orderbook

import (
	"errors"
	"time"
)

var (
	// MaxQueueSize is the max size of the queue
	MaxQueueSize int = 100
)

var (
	ErrBadOrderPrice       error = errors.New("price should be greater than 1")
	ErrBadOrderQty         error = errors.New("qty should be greater than 1")
	ErrTooLargeSizeOfQueue error = errors.New("too large size to create the queue")
	ErrDataNotFound        error = errors.New("data not found")
)

var OrderExpiration time.Duration = 86400 * time.Second // 1 day

// Side is the type of order side
type Side int

const (
	Buy Side = iota
	Sell
)

func (s Side) String() string {
	return [...]string{
		"buy",
		"sell",
	}[s]
}

// Option is an option type for OrderBook
type Option func(ob *OrderBook) error

// PriceMode is the mode of price of the order
type PriceMode int

const (
	Limit PriceMode = iota
	Market
	Unknown
)

func (p PriceMode) String() string {
	return [...]string{
		"limit",
		"market",
		"Unknown",
	}[p]
}

type OrderStatus int

const (
	StatusDone OrderStatus = iota
	StatusCompleted
	StatusPending
	StatusCanceled
	StatusUnknown
)

func (o OrderStatus) String() string {
	return [...]string{
		"done",
		"completed",
		"pending",
		"canceled",
		"unknown",
	}[o]
}

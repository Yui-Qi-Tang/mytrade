package orderbook

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewOrder returns new order
func NewOrder(side Side, price, qty int) (*Order, error) {
	if qty < 1 {
		return nil, ErrBadOrderQty
	}
	if price < 1 {
		return nil, ErrBadOrderPrice
	}
	o := &Order{ID: uuid.New(), Side: side, Price: price, Qty: qty, Time: time.Now(), PriceMode: Unknown}
	return o, nil
}

// Order is the structure of the order
type Order struct {
	ID        uuid.UUID `json:"id"`
	Price     int       `json:"price"`
	PriceMode PriceMode `json:"price_mode"`
	Qty       int       `json:"quantity"`
	Side      Side      `json:"side"`
	Time      time.Time `json:"time"`

	// idx is the index in the queue
	idx int
}

func (o Order) String() string {
	// order[id]:[side][price mode]-<price, qty, time, index>
	return fmt.Sprintf(
		"order[%s]:[%s][%s]-<[price]: %d, [qty]: %d, [index]: %d, [time]: %s>\n",
		o.ID.String(), o.Side.String(), o.PriceMode.String(), o.Price, o.Qty, o.idx, o.Time,
	)
}

// Orders is the queue of orders
type Orders []*Order

// Len implements interface of pkg container/heap which returns length of Orders
func (o Orders) Len() int { return len(o) }

// Less implements interface of pkg container/heap which returns priority of each element of the order queue
func (o Orders) Less(i, j int) bool {

	// if order i & j have the same price, just compare the time.
	// less timestamp value one has the high priority
	if o[i].Price == o[j].Price {
		return o[i].Time.UTC().UnixNano() < o[j].Time.UTC().UnixNano()
	}

	if o[i].Side == Buy {
		return o[i].Price > o[j].Price
	}
	return o[i].Price < o[j].Price
}

// Push implements interface of pkg container/heap which appends element into the order queue
func (o *Orders) Push(x any) {
	order := x.(*Order)
	order.idx = o.Len() // update index of new order
	*o = append(*o, order)
}

// Swap implements interface of pkg container/heap which exchanges elements i&j and update the index of the queue
func (o Orders) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
	o[i].idx = i
	o[j].idx = j
}

// Pop implements interface of pkg container/heap which pop the latest element from the orders
func (o *Orders) Pop() any {
	oldOrders := *o
	oldOrdersLen := oldOrders.Len()
	popOrder := oldOrders[oldOrdersLen-1]
	oldOrders[oldOrdersLen-1] = nil
	popOrder.idx = -1
	*o = oldOrders[0 : oldOrdersLen-1]
	return popOrder
}

// Dump shows each elements in the queue without priority possibly, this function is used in debug
// NOTE: this is not thread-safe, DON'T use this function in the production enviroment
func (o Orders) Dump() {
	for _, v := range o {
		fmt.Println(v)
	}
}

// update modifies the priority and value of an Item in the queue.
func (o *Orders) Fix(i int) {
	heap.Fix(o, i)
}

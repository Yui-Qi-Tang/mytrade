package orderbook

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// OrderBook is the main structure of orderbook
type OrderBook struct {
	sync.RWMutex
	// Done saves the orders are matched
	Done map[string]Order
	Bids Orders
	Asks Orders

	//maxQueueSize  int
	cleanTimeFreq time.Duration
}

// WithCleanTimeFrequecy is an option for the frequecy of the cleaning the expiration of the auto-cleaner
func WithCleanTimeFrequecy(duration time.Duration) Option {
	return func(ob *OrderBook) error {
		ob.cleanTimeFreq = duration
		return nil
	}
}

// New returns an Orderbook
func New(opts ...Option) (*OrderBook, error) {

	ob := &OrderBook{
		Done:          make(map[string]Order),
		Bids:          make(Orders, 0, MaxQueueSize),
		Asks:          make(Orders, 0, MaxQueueSize),
		cleanTimeFreq: 10 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(ob); err != nil {
			return nil, err
		}
	}
	return ob, nil
}

// Info prints the information of the orderbook
func (ob *OrderBook) Info() {
	bids, asks := ob.GetBids(), ob.GetAsks()
	log.Println("===> Orderbook settings...")
	fmt.Printf("[Bids] orders: %d, available: %d\n", len(bids), cap(bids))
	fmt.Printf("[Asks] orders: %d, available: %d\n", len(asks), cap(asks))
	fmt.Printf("[Complete Order]: %d\n", len(ob.GetCompleteOrders()))
	log.Println("... Orderbook information <===")
}

// GetBids returns bids orders
func (ob *OrderBook) GetBids() Orders {
	ob.RLock()
	defer ob.RUnlock()
	return ob.Bids
}

// GetAsks returns asks orders
func (ob *OrderBook) GetAsks() Orders {
	ob.RLock()
	defer ob.RUnlock()
	return ob.Asks
}

// CheckQueueSize checks the size is less than the MaxQueueSize
func (ob *OrderBook) CheckQueueSize(side Side) error {
	switch side {
	case Buy:
		if len(ob.GetBids()) >= MaxQueueSize {
			return ErrTooLargeSizeOfQueue
		}
	case Sell:
		if len(ob.GetAsks()) >= MaxQueueSize {
			return ErrTooLargeSizeOfQueue
		}
	default:
		// This is blank
	}
	return nil
}

// ProcessLimitOrder processes limit order and returns order id
func (ob *OrderBook) ProcessLimitOrder(side Side, price, qty int) (string, error) {
	if err := ob.CheckQueueSize(side); err != nil {
		return "", err
	}

	// create order
	newOrder, err := NewOrder(side, price, qty)
	if err != nil {
		return "", err
	}
	newOrder.PriceMode = Limit // set price mode
	// trade
	if err := ob.process(newOrder); err != nil {
		return "", nil
	}
	return newOrder.ID.String(), nil
}

// ProcessMarketOrder processes market order and returns order id
func (ob *OrderBook) ProcessMarketOrder(side Side, qty int) (string, error) {
	if err := ob.CheckQueueSize(side); err != nil {
		return "", err
	}
	// Hint: 1 is the default price if there is no 'price' before
	// should I make the default price configurable?
	newOrder, err := NewOrder(side, 1, qty)
	if err != nil {
		return "", err
	}
	newOrder.PriceMode = Market // set price mode

	// trade
	if err := ob.process(newOrder); err != nil {
		return "", nil
	}
	return newOrder.ID.String(), nil
}

// process process order
func (ob *OrderBook) process(o *Order) error {

	// copy the origin qty
	copyQty := o.Qty
	// trade
	if err := ob.Trade(o); err != nil {
		return err
	}
	// save the new order if complete
	if o.Qty == 0 {
		o.Qty = copyQty
		ob.save(o)
	} else { // otherwise, push this order to the queue
		ob.PushOrderSync(o)
	}
	return nil
}

// PushOrderSync pushes order into queue by side with lock
func (ob *OrderBook) PushOrderSync(o *Order) {
	ob.Lock()
	defer ob.Unlock()
	ob.PushOrder(o)
}

// PushOrder pushes order into the queue by side
func (ob *OrderBook) PushOrder(o *Order) {
	if o.Side == Buy {
		heap.Push(&ob.Bids, o)
	} else {
		heap.Push(&ob.Asks, o)
	}
}

// PopBySide pops order by side
func (ob *OrderBook) PopBySide(side Side) *Order {
	if side == Buy {
		return heap.Pop(&ob.Asks).(*Order)
	}
	return heap.Pop(&ob.Bids).(*Order)
}

// GetSideQueueLenSync returns length of queue by side with lock
func (ob *OrderBook) GetSideQueueLenSync(side Side) int {
	ob.RLock()
	defer ob.RUnlock()
	return ob.GetSideQueueLen(side)
}

// GetSideQueueLen returns length of queue by side
func (ob *OrderBook) GetSideQueueLen(side Side) int {
	if side == Buy {
		return ob.Asks.Len()
	}
	return ob.Bids.Len()
}

// cmp compares the relation of i and j by side
func (ob *OrderBook) cmp(side Side, i, j int) bool {
	if i == j {
		return true
	}
	if side == Buy {
		return i < j
	}
	return i > j
}

// Trade exchanges the order and the order from the side queue
func (ob *OrderBook) Trade(order *Order) error {
	// no seller so return
	if ob.GetSideQueueLenSync(order.Side) == 0 {
		return nil
	}

	ob.Lock()
	skips := make(Orders, 0)
	completes := make(Orders, 0)
	for ob.GetSideQueueLen(order.Side) > 0 && order.Qty > 0 {
		pop := ob.PopBySide(order.Side)
		popQty := pop.Qty // keep old version pop.qty

		// order follows the price from pop if the price mode of order is Market
		if order.PriceMode == Market {
			order.Price = pop.Price
		}

		// pop follows the price from order if the price mode of pop is Market
		if pop.PriceMode == Market {
			pop.Price = order.Price
		}

		// exchange
		if ob.cmp(order.Side, pop.Price, order.Price) {
			// exchange Qty
			if pop.Qty == order.Qty {
				pop.Qty = 0
				order.Qty = 0
			} else if pop.Qty > order.Qty {
				pop.Qty = pop.Qty - order.Qty
				order.Qty = 0
			} else {
				order.Qty = order.Qty - pop.Qty
				pop.Qty = 0
			}

			// push back pop if pop.Qty > 1 (pop order is not completely)
			if pop.Qty > 1 {
				ob.PushOrder(pop)
			}
			// collect the complete pop
			if popQty != pop.Qty {
				co := *pop
				co.Qty = popQty - co.Qty
				completes = append(completes, &co)
			}
		} else {
			// collect the skipped pop
			skips = append(skips, pop)
		}
	}
	// push back the skip order into queue
	for _, skip := range skips {
		ob.PushOrder(skip)
	}
	ob.Unlock()

	// saves the complete (pop) order
	ob.save(completes...)

	return nil
}

// AutoCleanOrderQueue is the routine for cleaning expiration
func (o *OrderBook) AutoCleanOrderQueue(ctx context.Context) {
	log.Println("auto cleaner is enabled")
	log.Printf("chek queues & done every %s\n", o.cleanTimeFreq)
	ticker := time.NewTicker(o.cleanTimeFreq)
	for {
		select {
		case <-ticker.C:
			o.cleanOldOrder()
		case <-ctx.Done():
			log.Println("orderbook: auto cleaner is leaving...")
			return
		}
	}
}

// cleanOldOrder cleans the expiration orders
func (o *OrderBook) cleanOldOrder() {
	o.Lock()
	defer o.Unlock()

	// check if order is expired in Bids
	for i, bid := range o.Bids {
		if time.Since(bid.Time) > OrderExpiration {
			o.Bids = append(o.Bids[:i], o.Bids[i+1:]...)
		}
	}

	// check if order is expired in Asks
	for i, bid := range o.Asks {
		if time.Since(bid.Time) > OrderExpiration {
			o.Asks = append(o.Asks[:i], o.Asks[i+1:]...)
		}
	}

	// check if order is expired in Done
	for k, v := range o.Done {
		if time.Since(v.Time) > OrderExpiration {
			delete(o.Done, k)
		}
	}

}

// GetOrder gets order by id and returns the status of the order
func (ob *OrderBook) GetOrder(id string, order *Order) (OrderStatus, error) {

	if order == nil {
		return StatusUnknown, errors.New("order is nil")
	}

	ob.RLock()
	defer ob.RUnlock()

	for _, bid := range ob.Bids {
		if bid.ID.String() == id {
			*order = *bid
			return StatusPending, nil
		}
	}

	for _, bid := range ob.Asks {
		if bid.ID.String() == id {
			*order = *bid
			return StatusPending, nil
		}
	}

	if v, exist := ob.Done[id]; exist {
		*order = v
		return StatusCompleted, nil
	}

	return StatusCanceled, nil
}

// save saves the completed orders
func (o *OrderBook) save(orders ...*Order) {
	o.Lock()
	defer o.Unlock()
	for _, order := range orders {
		oid := order.ID.String()
		if v, exist := o.Done[oid]; exist {
			v.Price = order.Price
			v.Qty += order.Qty
			o.Done[oid] = v
		} else {
			o.Done[oid] = *order
		}
	}
}

// GetCompleteOrders returns complete orders
func (o *OrderBook) GetCompleteOrders() map[string]Order {
	o.RLock()
	defer o.RUnlock()
	return o.Done
}

// GetCompleteOrder returns complete order by id
func (o *OrderBook) GetCompleteOrder(id string, order *Order) error {
	o.RLock()
	defer o.RUnlock()

	v, exist := o.Done[id]
	if !exist {
		return ErrDataNotFound
	}
	*order = v
	return nil
}

/* version 1 for order exchange
// TradeLimitBids trades the bid and ask orders with limit price
func (ob *OrderBook) TradeLimitBids(order *Order) error {
	// no seller so return
	if ob.GetSideQueueLenSync(order.Side) == 0 {
		return nil
	}

	ob.Lock()
	buf := make(Orders, 0)
	completes := make(Orders, 0)
	for ob.Asks.Len() > 0 && order.Qty > 0 {
		ask := ob.PopBySide(order.Side)
		originASKQty := ask.Qty

		if order.PriceMode == Market {
			order.Price = ask.Price
		}

		if ask.PriceMode == Market {
			ask.Price = order.Price
		}

		//if ask.Price <= order.Price {
		if ob.cmp(order.Side, ask.Price, order.Price) {
			if ask.Qty == order.Qty {
				ask.Qty = 0
				order.Qty = 0
			} else if ask.Qty > order.Qty {
				ask.Qty = ask.Qty - order.Qty
				order.Qty = 0
			} else {
				order.Qty = order.Qty - ask.Qty
				ask.Qty = 0
			}

			if ask.Qty > 1 {
				ob.PushOrder(ask)
			}
			// save the complete ask
			if originASKQty != ask.Qty {
				co := *ask
				co.Qty = originASKQty - co.Qty
				completes = append(completes, &co)
			}
		} else {
			buf = append(buf, ask)
		}
	}
	for _, b := range buf {
		ob.PushOrder(b)
	}
	ob.Unlock()

	ob.save(completes...)

	return nil
}
*/

/* version 1 for order exchange
// TradeLimitAsk trades ask order and bid orders with limit price
func (ob *OrderBook) TradeLimitAsks(order *Order) error {
	// no buyer so return
	if len(ob.GetBids()) == 0 {
		return nil
	}

	ob.Lock()
	buf := make(Orders, 0)
	completes := make(Orders, 0)
	for ob.Bids.Len() > 0 && order.Qty > 0 {
		bid := heap.Pop(&ob.Bids).(*Order)
		originBidQty := bid.Qty

		if order.PriceMode == Market {
			order.Price = bid.Price
		}

		if bid.PriceMode == Market {
			bid.Price = order.Price
		}

		if bid.Price >= order.Price {
			if bid.Qty == order.Qty {
				bid.Qty = 0
				order.Qty = 0
			} else if bid.Qty > order.Qty {
				bid.Qty = bid.Qty - order.Qty
				order.Qty = 0
			} else {
				order.Qty = order.Qty - bid.Qty
				bid.Qty = 0
			}

			if bid.Qty > 1 {
				ob.PushOrder(bid)
			}

			// save the complete bid
			if originBidQty != bid.Qty {
				co := *bid
				co.Qty = originBidQty - co.Qty
				completes = append(completes, &co)
			}

		} else {
			buf = append(buf, bid)
		}
	}

	for _, b := range buf {
		ob.PushOrder(b)
	}

	ob.Unlock()

	ob.save(completes...)
	return nil
}
*/

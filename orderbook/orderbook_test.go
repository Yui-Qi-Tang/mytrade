package orderbook

import (
	"context"
	"testing"
	"time"
)

func TestOrderBook(t *testing.T) {
	t.Log("start testing orderbook...")
	ob, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// add limit order
	go func() {
		// add order in async. way
		ob.ProcessLimitOrder(Buy, 100, 10)
	}()

	id, err := ob.ProcessLimitOrder(Sell, 100, 10)
	if err != nil {
		t.Fatal(err)
	}

	var order Order
	// wait for the orders trading completely
	for {
		status, err := ob.GetOrder(id, &order)
		if err != nil {
			t.Fatal(err)
		}

		if status == StatusCompleted {
			break
		}
	}

	// add order with market type
	mOrderID, err := ob.ProcessMarketOrder(Buy, 10)
	if err != nil {
		t.Fatal(err)
	}
	// check if the order is in the pending
	mIdStatus, err := ob.GetOrder(mOrderID, &order)
	if err != nil {
		t.Fatal(err)
	}
	if mIdStatus != StatusPending {
		t.Fatalf("the status of order[%s] should be: %s, but got %s", order.ID, StatusPending, mIdStatus)
	}

	ob.ProcessMarketOrder(Sell, 5)
	ob.ProcessMarketOrder(Sell, 5)

	// the status of the same order should be completely
	mIdStatus, err = ob.GetOrder(mOrderID, &order)
	if err != nil {
		t.Fatal(err)
	}

	if mIdStatus != StatusCompleted {
		t.Fatalf("the status of order[%s] should be: %s, but got %s", order.ID, StatusCompleted, mIdStatus)
	}

	// check the number of the done records
	numOfCompleteOrders := len(ob.GetCompleteOrders())
	if numOfCompleteOrders != 5 {
		t.Fatalf("the number of complete orders should be: %d, but got %d", 5, numOfCompleteOrders)
	}
}

func TestOrderBookQTY(t *testing.T) {
	t.Log("start testing orderbook QTY...")

	testcases := []struct {
		orders []struct {
			side  Side
			pm    PriceMode
			price int
			qty   int
		}
		want int
	}{
		{
			orders: []struct {
				side  Side
				pm    PriceMode
				price int
				qty   int
			}{
				{side: Buy, pm: Limit, price: 100, qty: 10}, // focus on first order
				{side: Sell, pm: Limit, price: 100, qty: 1},
				{side: Sell, pm: Limit, price: 101, qty: 1},
			},
			want: 1,
		},
		{
			orders: []struct {
				side  Side
				pm    PriceMode
				price int
				qty   int
			}{
				{side: Sell, pm: Limit, price: 100, qty: 10}, // focus on first order
				{side: Buy, pm: Limit, price: 100, qty: 1},
				{side: Buy, pm: Limit, price: 101, qty: 1},
			},
			want: 2,
		},
		{
			orders: []struct {
				side  Side
				pm    PriceMode
				price int
				qty   int
			}{
				{side: Buy, pm: Market, price: 1, qty: 10}, // focus on first order
				{side: Sell, pm: Limit, price: 100, qty: 1},
				{side: Sell, pm: Limit, price: 101, qty: 1},
				{side: Sell, pm: Limit, price: 200, qty: 8},
			},
			want: 10,
		},
		{
			orders: []struct {
				side  Side
				pm    PriceMode
				price int
				qty   int
			}{
				{side: Sell, pm: Market, price: 1, qty: 10}, // focus on first order
				{side: Buy, pm: Limit, price: 10, qty: 2},
				{side: Buy, pm: Limit, price: 1000, qty: 1},
				{side: Buy, pm: Limit, price: 299, qty: 6},
			},
			want: 9,
		},
	}

	for _, tt := range testcases {
		ob, err := New()
		if err != nil {
			t.Fatal("failed to create orderbook", err)
		}

		var targetID string = ""
		// generate orders
		for i, o := range tt.orders {
			switch o.pm {
			case Limit:
				id, err := ob.ProcessLimitOrder(o.side, o.price, o.qty)
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					targetID = id
				}
			case Market:

				id, err := ob.ProcessMarketOrder(o.side, o.qty)
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					targetID = id
				}
			}
			if len(targetID) == 0 {
				t.Fatal("targetID is empty")
			}
		}

		// get complete order
		var cOrder Order
		if err := ob.GetCompleteOrder(targetID, &cOrder); err != nil {
			t.Fatal(err)
		}
		if cOrder.Qty != tt.want {
			t.Fatalf("it should be %d, but got %d", tt.want, cOrder.Qty)
		}
	}

	t.Log("... Passed")

}

func TestMaxSizeOfQueue(t *testing.T) {

	t.Log("start testing the limitation of queue size...")

	ob, err := New()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < MaxQueueSize; i++ {
		if _, err := ob.ProcessLimitOrder(Buy, 100, 10); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := ob.ProcessLimitOrder(Buy, 100, 10); err != ErrTooLargeSizeOfQueue {
		t.Fatal("wrong error type", err)
	}

	for i := 0; i < MaxQueueSize; i++ {
		ob.ProcessLimitOrder(Sell, 101, 10)
	}

	if _, err := ob.ProcessLimitOrder(Sell, 101, 10); err != ErrTooLargeSizeOfQueue {
		t.Fatal("wrong error type", err)
	}

	t.Log("... Passed")

}

func TestDone(t *testing.T) {

	t.Log("start testing Done records...")

	ob, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// add buy with price: 100, qty: 10
	for i := 0; i < MaxQueueSize; i++ {
		if _, err := ob.ProcessLimitOrder(Buy, 100, 10); err != nil {
			t.Fatal(err)
		}
	}

	// add sell with price: 100, qty: 10
	for i := 0; i < MaxQueueSize; i++ {
		if _, err := ob.ProcessLimitOrder(Sell, 100, 10); err != nil {
			t.Fatal(err)
		}
	}

	if ob.GetAsks().Len() > 0 {
		t.Fatal("wrong size of asks:", ob.GetAsks().Len(), "it should be zero")
	}

	if ob.GetBids().Len() > 0 {
		t.Fatal("wrong size of bids:", ob.GetBids().Len(), "it should be zero")
	}

	if len(ob.Done) != 200 {
		t.Fatal("wrong size of bids:", len(ob.Done), "it should be 200")
	}

	t.Log("... Passed")

}

func TestAutoCleanQueue(t *testing.T) {

	t.Log("start testing clean expired order automatically...")

	originOE := OrderExpiration
	OrderExpiration = 100 * time.Millisecond // updated the expiration directly
	ob, err := New(WithCleanTimeFrequecy(1 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	// first add 2 orders
	ob.ProcessLimitOrder(Buy, 100, 10)
	ob.ProcessLimitOrder(Sell, 120, 10)

	if len(ob.GetAsks()) != 1 {
		t.Fatal("the size of asks should be 1")
	}
	if len(ob.GetBids()) != 1 {
		t.Fatal("the size of bids should be 1")
	}

	// enable auto clean queue, for test reason, I enable that after orders are enqueued
	ctx, cancel := context.WithCancel(context.Background())
	go ob.AutoCleanOrderQueue(ctx)
	time.Sleep(500 * time.Millisecond) // wait for cleaner

	if len(ob.GetAsks()) != 0 {
		t.Fatal("the size of asks should be 0")
	}
	if len(ob.GetBids()) != 0 {
		t.Fatal("the size of bids should be 0")
	}
	cancel() // let the auto cleaner leave

	OrderExpiration = originOE // resume the default setting of OrderExpiration
	t.Log("... Passed")
}

func TestTradePrice(t *testing.T) {

	t.Log("start testing price of the trading...")

	ob, err := New()
	if err != nil {
		t.Fatal()
	}

	// 'market Sell' with price 1 because there is no price before
	id, err := ob.ProcessMarketOrder(Sell, 10)
	if err != nil {
		t.Fatal(err)
	}

	// 'market Buy' with price 1 because there is no price before
	if _, err := ob.ProcessMarketOrder(Buy, 5); err != nil {
		t.Fatal(err)
	}

	var order Order
	if err := ob.GetCompleteOrder(id, &order); err != nil {
		t.Fatal(err)
	}

	if order.Price != 1 {
		t.Fatal("the price should be 1, becuase there is no exchage before")
	}

	// 'limit Buy'
	price := 100
	if _, err := ob.ProcessLimitOrder(Buy, price, 5); err != nil {
		t.Fatal(err)
	}

	if err := ob.GetCompleteOrder(id, &order); err != nil {
		t.Fatal(err)
	}

	if order.Price != price {
		t.Fatalf("the price should be %d, becuase there is an order with limited price ", price)
	}

	t.Log("... Passed")

}

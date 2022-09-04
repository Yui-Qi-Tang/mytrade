package orderbook

import (
	"container/heap"
	"testing"
)

func TestSequenceOfBuyerOrders(t *testing.T) {

	// seq of bids => for i, j in bids, bids[i] > bids[j]
	testcases := []struct {
		inputs []struct {
			s          Side
			price, qty int
		}
		want int
	}{
		{
			inputs: []struct {
				s          Side
				price, qty int
			}{
				{
					s:     Buy,
					price: 100,
					qty:   10,
				},
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 100,
					qty:   10,
				},
				{
					s:     Buy,
					price: 100,
					qty:   10,
				},
			},
			want: 100,
		},
		{
			inputs: []struct {
				s          Side
				price, qty int
			}{
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 10,
					qty:   10,
				},
				{
					s:     Buy,
					price: 200,
					qty:   10,
				},
				{
					s:     Buy,
					price: 100,
					qty:   10,
				},
			},
			want: 200,
		},
	}

	for _, tt := range testcases {
		orders := make(Orders, 0, len(tt.inputs))
		for _, in := range tt.inputs {
			order, err := NewOrder(in.s, in.price, in.qty)
			if err != nil {
				t.Fatal(err)
			}
			orders.Push(order)
		}
		heap.Init(&orders)

		head := heap.Pop(&orders).(*Order)
		if head.Price != tt.want {
			t.Fatalf("it should be %d, but got %d", tt.want, head.Price)
		}
	}
}

func TestSequenceOfSellerOrders(t *testing.T) {
	// seq of asks => for i, j in asks, asks[i] < bids[j]
	testcases := []struct {
		inputs []struct {
			s          Side
			price, qty int
		}
		want int
	}{
		{
			inputs: []struct {
				s          Side
				price, qty int
			}{
				{
					s:     Sell,
					price: 100,
					qty:   10,
				},
				{
					s:     Sell,
					price: 10,
					qty:   10,
				},
				{
					s:     Sell,
					price: 10,
					qty:   10,
				},
				{
					s:     Sell,
					price: 10,
					qty:   10,
				},
				{
					s:     Sell,
					price: 100,
					qty:   10,
				},
				{
					s:     Sell,
					price: 100,
					qty:   10,
				},
			},
			want: 10,
		},
		{
			inputs: []struct {
				s          Side
				price, qty int
			}{
				{
					s:     Sell,
					price: 12,
					qty:   10,
				},
				{
					s:     Sell,
					price: 1,
					qty:   10,
				},
				{
					s:     Sell,
					price: 15,
					qty:   10,
				},
				{
					s:     Sell,
					price: 200,
					qty:   10,
				},
				{
					s:     Sell,
					price: 100,
					qty:   10,
				},
			},
			want: 1,
		},
	}

	for _, tt := range testcases {
		orders := make(Orders, 0, len(tt.inputs))
		for _, in := range tt.inputs {
			order, err := NewOrder(in.s, in.price, in.qty)
			if err != nil {
				t.Fatal(err)
			}
			orders.Push(order)
		}
		heap.Init(&orders)

		head := heap.Pop(&orders).(*Order)
		if head.Price != tt.want {
			t.Fatalf("it should be %d, but got %d", tt.want, head.Price)
		}
	}
}

func TestCreateInvalidPriceOrQtyOrder(t *testing.T) {
	testcases := []struct {
		badPrice, badQty int
		want             error
	}{
		{ // bad price, valid qty
			badPrice: -1,
			badQty:   10,
			want:     ErrBadOrderPrice,
		},
		{ // bad qty, valid price
			badPrice: 10,
			badQty:   -1,
			want:     ErrBadOrderQty,
		},
		{ // bad qty & price
			badPrice: -1,
			badQty:   -1,
			want:     ErrBadOrderQty,
		},
	}

	for _, tt := range testcases {
		_, err := NewOrder(Buy, tt.badPrice, tt.badQty)
		if err != tt.want {
			t.Fatalf("the error should be %v, but got %v", tt.want, err)
		}
	}
}

func TestSeqOfSamePriceOrder(t *testing.T) {
	testcases := []struct {
		s      Side
		prices []int
	}{
		{
			s:      Buy,
			prices: []int{3, 3},
		},
		{
			s:      Buy,
			prices: []int{3, 1, 3},
		},
		{
			s:      Sell,
			prices: []int{1, 3, 1},
		},
	}

	for _, tt := range testcases {
		orders := make(Orders, 0, len(tt.prices))
		for _, price := range tt.prices {
			order, err := NewOrder(tt.s, price, 1)
			if err != nil {
				t.Fatal(err)
			}
			orders.Push(order)
		}
		// just pick first two elements
		o1 := heap.Pop(&orders).(*Order)
		o2 := heap.Pop(&orders).(*Order)

		if o1.Price != o2.Price {
			t.Fatalf("wrong priority of side: %s", tt.s)
		}

		if o1.Time.UnixNano() > o2.Time.UnixNano() {
			t.Fatalf("wrong priority of side: %s", tt.s)
		}
	}
}

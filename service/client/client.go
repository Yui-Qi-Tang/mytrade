package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"mytrader.github.com/orderbook"
	pb "mytrader.github.com/service/protoc"
)

var orderBookSide = map[string]orderbook.Side{
	"buy":  orderbook.Buy,
	"sell": orderbook.Sell,
}

var orderBookPriceMode = map[string]orderbook.PriceMode{
	"limit":  orderbook.Limit,
	"market": orderbook.Market,
}

func main() {

	var (
		serverAddr string
		call       string
		qty        int64
		price      int64

		side      string
		priceMode string

		oid string
	)

	flag.StringVar(&serverAddr, "server-addr", "localhost:9999", "the address of the server")
	flag.StringVar(&call, "call", "", "call for server [create_order|get_order]")
	flag.StringVar(&oid, "order_id", "", "order id")
	flag.Int64Var(&qty, "quantity", -1, "quantity of the the order")
	flag.StringVar(&side, "side", "", "side of the order [buy|sell]")
	flag.Int64Var(&price, "price", -1, "price of the order")
	flag.StringVar(&priceMode, "price_mode", "", "price mode of the order [market|limit]")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewTraderClient(conn)

	switch call {
	case "create_order":

		o, err := createTradeOrder(side, priceMode, int64(price), int64(qty))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		reply, err := client.Create(ctx, o)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		printReply(reply)

	case "get_order":
		if len(oid) == 0 {
			fmt.Println("id is empty")
			os.Exit(0)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		reply, err := client.Get(ctx, &pb.GetOrder{Id: oid})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		printReply(reply)

	default:
		fmt.Println("unkonwn command [create_order, ger_order]", call)
		os.Exit(0)
	}

}

func printReply(reply *pb.OrderReply) {
	log.Println("response from server => ")
	fmt.Println("order_id:", reply.ID)
	fmt.Println("timestamp:", reply.Timestamp)
	fmt.Printf("side: %s, price mode: %s\n", reply.Side, reply.PriceMode)
	fmt.Printf("price: %d, quantity: %d\n", reply.Price, reply.Quantity)
	fmt.Println("status:", reply.Status)
}

func createTradeOrder(side, priceMode string, price, qty int64) (*pb.Order, error) {

	s, exist := orderBookSide[side]
	if !exist {
		return nil, errors.New("bad side value, it should be buy or sell")
	}

	pm, exist := orderBookPriceMode[priceMode]
	if !exist {
		return nil, errors.New("bad price_mode value, it should be market or limit")
	}

	if pm == orderbook.Limit && price < 1 {
		return nil, errors.New("price should be >= 1")
	}
	if qty < 1 {
		return nil, errors.New("quantity should be >= 1")
	}

	o := &pb.Order{Side: int32(s), PriceMode: int32(pm), Price: price, Quantity: qty}
	return o, nil
}

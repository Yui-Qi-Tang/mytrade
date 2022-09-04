package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"mytrader.github.com/orderbook"
	"mytrader.github.com/service/server"
)

var GitCommit string // git describe --always

func main() {

	// clean freq.,  queue size, server addr and order expiration
	var (
		cleanOrderFreq int64
		maxQueueSize   int
		serverAddr     string
		orderExpired   int64
		version        bool
	)

	flag.Int64Var(&cleanOrderFreq, "clean_order_freq", 10, "freq. of auto clean order in second")
	flag.IntVar(&maxQueueSize, "max_queue_size", 100, "max. size of queue")
	flag.StringVar(&serverAddr, "listen_addr", "localhost:9999", "address of the server")
	flag.Int64Var(&orderExpired, "order_expired", 86400, "expiration of the order, this is used by auto cleaner")
	flag.BoolVar(&version, "version", false, "show version")
	flag.Parse()

	// version
	if version {
		appName := "mytrader"
		release := fmt.Sprintf("%s/%s, %s, %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version(), GitCommit)
		version := "0.1"
		fmt.Println(appName)
		fmt.Println(version)
		fmt.Println(release)
		return
	}

	// setup orderbook
	orderbook.OrderExpiration = time.Duration(orderExpired) * time.Second
	orderbook.MaxQueueSize = maxQueueSize
	ob, err := orderbook.New(orderbook.WithCleanTimeFrequecy(time.Duration(cleanOrderFreq) * time.Second))
	if err != nil {
		panic(err)
	}

	// setup server
	s, err := server.New(server.WithOrderBook(ob), server.WithAddr(serverAddr))
	if err != nil {
		panic(err)
	}

	if err := s.Run(); err != nil {
		panic(err)
	}

	log.Println("service is stopped")
}

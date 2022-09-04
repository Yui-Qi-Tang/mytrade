# mytrader

This is a very simple order match server.

# Project layout

`main.go`: is the main of this project

`orderbook/`: the orderbook pkg

`service/`: the service folder
  - `mytrader.proto`: is the gRPC spec.
  - `server/`: service server pkg
  - `client/`: service client 
  - `protoc/`: the spec. of the gRPC server interfaces and protobuf

# Run

1. testing and build client and server: `make`

2. Server: `bin/mytrader` (show options: `bin/mytrader -h`)

3. Client: `bin/mytrader-client` (show options: `bin/mytrader-client -h`)
    - create order: `bin/mytrader-client -call create_order -side $SIDE -price_mode $PRICEMODE -price 100 -quantity 50`
  where `$SIDE` = { buy | sell } and `$PRICEMODE` = { market | limit}

       - create an order with side: `sell`, price_mode: `market`, quantity: 50: `bin/mytrader-client -call create_order -side sell -price_mode market -quantity 50`
          - if your price_mode is `market`, the server will ingore the value of `price`
          - if your price_mode is `market`, and there is only the order in the system then the price of the reply will be -1
          - example reply:
          ```shell
            2022/09/04 19:41:32 response from server => 
            order_id: 4366f1be-c144-4878-8b99-c5b36c7654e2
            timestamp: 1662291692
            side: sell, price mode: market
            price: 100, quantity: 50
            status: pending
          ```

    - get_order: `bin/mytrader-client -call get_order -order_id $ORDERID`
      - example reply:
      ```shell
         2022/09/04 19:44:18 response from server => 
         order_id: 60e72f24-a75a-4462-92b4-d8ea768004fd
         timestamp: 1662291558
         side: sell, price mode: market
         price: 1000, quantity: 50
         status: completed
      ```  

# Order Status

- pending: the order is still in the queue for trading
- completed: the order is successed to trade

# Implementations

- Server: gRPC, the spec. is put in `service/mytrader.proto`.
- Client: gRPC, the spec. is put in `service/mytrader.proto`.
- Queue: Priority Queue which is based on `container/heap`.
- Order canceled: order is canceled by auto-cleaner if the order is expired
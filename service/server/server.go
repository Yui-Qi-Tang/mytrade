package server

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"mytrader.github.com/orderbook"
	"mytrader.github.com/service/protoc"
)

type Option func(s *Server) error

type serveErr string

func (s serveErr) String() string {
	return string(s)
}

func (s serveErr) Signal() {
	// This blank
}

func WithAddr(addr string) Option {
	return func(s *Server) error {

		if len(addr) == 0 {
			return errors.New("the addr is empty")
		}

		s.addr = addr
		return nil
	}
}

// TODO: create an orderbook interface to decuple the server & orderbook
// for now, just use the mytrader.github.com/orderbook
func WithOrderBook(ob *orderbook.OrderBook) Option {
	return func(s *Server) error {

		if ob == nil {
			return errors.New("the orderbook is empty")
		}

		s.ob = ob
		return nil
	}
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		addr: "localhost:9999",
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

type Server struct {
	addr string
	ob   *orderbook.OrderBook
	protoc.UnimplementedTraderServer
}

func (s *Server) Create(ctx context.Context, order *protoc.Order) (*protoc.OrderReply, error) {

	var side orderbook.Side = orderbook.Side(order.Side)
	reply := &protoc.OrderReply{
		Price: order.Price,
	}

	switch orderbook.PriceMode(order.PriceMode) {
	case orderbook.Limit:
		lid, err := s.ob.ProcessLimitOrder(side, int(order.Price), int(order.Quantity))
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		reply.ID = lid
	case orderbook.Market:
		mid, err := s.ob.ProcessMarketOrder(side, int(order.Quantity))
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		reply.ID = mid
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown price mode")
	}

	s.ob.Info()

	var o orderbook.Order
	ostatus, err := s.ob.GetOrder(reply.ID, &o)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	reply.PriceMode = o.PriceMode.String()
	reply.Status = ostatus.String()
	reply.Side = o.Side.String()
	reply.Quantity = int64(o.Qty)
	reply.Timestamp = o.Time.Unix()
	return reply, nil

}

func (s *Server) Get(ctx context.Context, order *protoc.GetOrder) (*protoc.OrderReply, error) {
	var o orderbook.Order
	ostatus, err := s.ob.GetOrder(order.Id, &o)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	reply := &protoc.OrderReply{
		ID:        o.ID.String(),
		Side:      o.Side.String(),
		Price:     int64(o.Price),
		PriceMode: o.PriceMode.String(),
		Status:    ostatus.String(),
		Quantity:  int64(o.Qty),
		Timestamp: o.Time.Unix(),
	}
	return reply, nil
}

func (s *Server) Run() error {

	ctx, cancel := context.WithCancel(context.TODO())
	go s.ob.AutoCleanOrderQueue(ctx)
	defer cancel()

	s.ob.Info()

	log.Println("server runs at", s.addr)
	log.Printf("[Max. size of the queue]: %d\n", orderbook.MaxQueueSize)
	log.Printf("[Order live time]: %v\n", orderbook.OrderExpiration)

	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	gs := grpc.NewServer()
	protoc.RegisterTraderServer(gs, s)

	// for gracful shutdown
	var se serveErr
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, se)

	go func() {
		if err := gs.Serve(lis); err != nil {
			se = serveErr(err.Error())
			shutdown <- se
		}
	}()

	// gracefull shutdown
	sd := <-shutdown
	cancel()

	if e, ok := sd.(serveErr); !ok {
		gs.Stop()
		log.Println("server: bye!")
	} else {
		return errors.New("server can not serve, because: " + e.String())
	}
	return nil

}

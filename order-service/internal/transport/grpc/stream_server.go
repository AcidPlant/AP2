package grpc

import (
	"log"
	"time"

	"order-service/internal/broker"
	"order-service/internal/repository"

	orderv1 "github.com/AcidPlant/generated-code/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderStreamServer struct {
	orderv1.UnimplementedOrderServiceServer
	repo   repository.OrderRepository
	broker *broker.OrderBroker
}

func NewOrderStreamServer(repo repository.OrderRepository, b *broker.OrderBroker) *OrderStreamServer {
	return &OrderStreamServer{repo: repo, broker: b}
}

func (s *OrderStreamServer) SubscribeToOrderUpdates(
	req *orderv1.OrderRequest,
	stream orderv1.OrderService_SubscribeToOrderUpdatesServer,
) error {
	orderID := req.GetOrderId()
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.repo.GetByID(stream.Context(), orderID)
	if err != nil {
		return status.Errorf(codes.Internal, "fetch order: %v", err)
	}
	if order == nil {
		return status.Errorf(codes.NotFound, "order %s not found", orderID)
	}

	if err := stream.Send(&orderv1.OrderStatusUpdate{
		OrderId:   order.ID,
		Status:    order.Status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	if isTerminal(order.Status) {
		return nil
	}

	ch := s.broker.Subscribe(orderID)
	defer s.broker.Unsubscribe(orderID, ch)

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			log.Printf("streaming update: order=%s status=%s", event.OrderID, event.Status)
			if err := stream.Send(&orderv1.OrderStatusUpdate{
				OrderId:   event.OrderID,
				Status:    event.Status,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}); err != nil {
				return err
			}
			if isTerminal(event.Status) {
				return nil
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func isTerminal(s string) bool {
	return s == "Paid" || s == "Failed" || s == "Cancelled"
}

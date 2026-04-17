package grpc

import (
	"context"
	"errors"
	"time"

	"payment-service/internal/usecase"

	paymentv1 "github.com/AcidPlant/generated-code/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentGRPCServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	uc usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *paymentv1.PaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	payment, err := s.uc.Authorize(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "authorize payment: %v", err)
	}

	return &paymentv1.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        payment.Status,
		ProcessedAt:   timestamppb.New(time.Now().UTC()),
	}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	minAmt := req.GetMinAmount()
	maxAmt := req.GetMaxAmount()

	if minAmt < 0 || maxAmt < 0 {
		return nil, status.Error(codes.InvalidArgument, "min_amount and max_amount must be non-negative")
	}

	payments, err := s.uc.ListPayments(ctx, minAmt, maxAmt)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidRange) {
			return nil, status.Error(codes.InvalidArgument, "min_amount cannot be greater than max_amount")
		}
		return nil, status.Errorf(codes.Internal, "list payments: %v", err)
	}

	items := make([]*paymentv1.PaymentItem, 0, len(payments))
	for _, p := range payments {
		items = append(items, &paymentv1.PaymentItem{
			Id:            p.ID,
			OrderId:       p.OrderID,
			TransactionId: p.TransactionID,
			Amount:        p.Amount,
			Status:        p.Status,
		})
	}

	return &paymentv1.ListPaymentsResponse{
		Payments: items,
	}, nil
}

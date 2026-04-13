package client

import (
	"context"
	"fmt"

	paymentv1 "github.com/AcidPlant/generated-code/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// PaymentGRPCClient satisfies the PaymentClient interface defined in usecase.
// The interface in order_usecase.go is UNCHANGED — only this delivery layer changes.
type PaymentGRPCClient struct {
	client paymentv1.PaymentServiceClient
}

func NewPaymentGRPCClient(grpcAddr string) (*PaymentGRPCClient, error) {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial payment-service: %w", err)
	}
	return &PaymentGRPCClient{client: paymentv1.NewPaymentServiceClient(conn)}, nil
}

// Authorize satisfies the usecase.PaymentClient interface — signature is identical to before.
func (c *PaymentGRPCClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	resp, err := c.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.Unavailable {
			return "", "", fmt.Errorf("payment service unreachable: %w", err)
		}
		return "", "", fmt.Errorf("process payment rpc: %w", err)
	}
	return resp.GetTransactionId(), resp.GetStatus(), nil
}

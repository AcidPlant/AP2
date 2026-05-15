package client

import (
	"context"
	"fmt"
	"sync"

	paymentv1 "github.com/AcidPlant/generated-code/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const maxRetry = 5

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

func (c *PaymentGRPCClient) Authorize(ctx context.Context, orderID string, amount int64, customerEmail string) (string, string, error) {
	type result struct {
		txID   string
		status string
		err    error
	}

	results := make(chan result, maxRetry)

	md := metadata.Pairs("customer-email", customerEmail)
	outCtx := metadata.NewOutgoingContext(ctx, md)

	var wg sync.WaitGroup
	for i := 0; i < maxRetry; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.client.ProcessPayment(outCtx, &paymentv1.PaymentRequest{
				OrderId: orderID,
				Amount:  amount,
			})
			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() == codes.Unavailable {
					results <- result{err: fmt.Errorf("payment service unreachable: %w", err)}
					return
				}
				results <- result{err: fmt.Errorf("process payment rpc: %w", err)}
				return
			}
			results <- result{txID: resp.GetTransactionId(), status: resp.GetStatus()}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var lastErr error
	for r := range results {
		if r.err == nil {
			return r.txID, r.status, nil
		}
		lastErr = r.err
	}
	return "", "", lastErr
}

package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type MockProvider struct {
	latency     time.Duration
	failureRate float64
}

func NewMockProvider(latencyMs int, failureRate float64) NotificationProvider {
	if latencyMs <= 0 {
		latencyMs = 200
	}
	if failureRate < 0 || failureRate > 1 {
		failureRate = 0.3
	}
	return &MockProvider{
		latency:     time.Duration(latencyMs) * time.Millisecond,
		failureRate: failureRate,
	}
}

func (p *MockProvider) Send(ctx context.Context, n Notification) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.latency):
	}

	if rand.Float64() < p.failureRate {
		return errors.New("mock provider: transient network error (simulated)")
	}

	dollars := fmt.Sprintf("$%.2f", float64(n.Amount)/100.0)
	log.Printf("[MockProvider] ✉  Email sent  to=%s  order=%s  amount=%s  status=%s",
		n.To, n.OrderID, dollars, n.Status)
	return nil
}

package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"notification-service/internal/idempotency"
	"notification-service/internal/provider"
)

type Config struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxRetries: 4,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type NotificationWorker struct {
	idem     *idempotency.RedisStore
	provider provider.NotificationProvider
	cfg      Config
}

func New(idem *idempotency.RedisStore, p provider.NotificationProvider, cfg Config) *NotificationWorker {
	return &NotificationWorker{idem: idem, provider: p, cfg: cfg}
}

func (w *NotificationWorker) Process(ctx context.Context, event PaymentEvent) error {
	acquired, err := w.idem.TryAcquire(ctx, event.EventID)
	if err != nil {
		log.Printf("[Worker] WARN idempotency check failed for event=%s: %v – proceeding", event.EventID, err)
	} else if !acquired {
		return nil
	}

	notif := provider.Notification{
		To:      event.CustomerEmail,
		OrderID: event.OrderID,
		Amount:  event.Amount,
		Status:  event.Status,
	}

	var lastErr error
	for attempt := 0; attempt < w.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := w.calcDelay(attempt)
			log.Printf("[Worker] RETRY attempt=%d/%d event=%s delay=%s err=%v",
				attempt, w.cfg.MaxRetries-1, event.EventID, delay, lastErr)
			select {
			case <-ctx.Done():
				return fmt.Errorf("worker: context cancelled during backoff: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		sendErr := w.provider.Send(ctx, notif)
		if sendErr == nil {
			if markErr := w.idem.MarkDone(ctx, event.EventID); markErr != nil {
				log.Printf("[Worker] WARN could not mark event=%s as done: %v", event.EventID, markErr)
			}
			log.Printf("[Worker] SUCCESS event=%s attempt=%d", event.EventID, attempt+1)
			return nil
		}
		lastErr = sendErr
		log.Printf("[Worker] send failed event=%s attempt=%d/%d: %v",
			event.EventID, attempt+1, w.cfg.MaxRetries, sendErr)
	}

	return fmt.Errorf("worker: all %d attempts failed for event=%s: %w", w.cfg.MaxRetries, event.EventID, lastErr)
}

func (w *NotificationWorker) calcDelay(attempt int) time.Duration {
	multiplier := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(w.cfg.BaseDelay) * multiplier)
	if delay > w.cfg.MaxDelay {
		delay = w.cfg.MaxDelay
	}
	return delay
}

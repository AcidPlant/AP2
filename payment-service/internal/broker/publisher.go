package broker

import "context"

type PaymentEvent struct {
	EventID       string `json:"event_id"` // unique per publish – used for idempotency
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type Publisher interface {
	Publish(ctx context.Context, event PaymentEvent) error
	Close() error
}

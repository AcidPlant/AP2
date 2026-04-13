package domain

import "time"

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

type Order struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
	ItemName       string    `json:"item_name"`
	Amount         int64     `json:"amount"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

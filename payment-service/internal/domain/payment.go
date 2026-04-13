package domain

const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"

	// MaxAmount is 100 000 cents ($1 000.00). Payments above this are declined.
	MaxAmount = int64(100000)
)

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

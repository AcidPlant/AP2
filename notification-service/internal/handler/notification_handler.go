package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"notification-service/internal/idempotency"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type NotificationHandler struct {
	idem *idempotency.Store
}

func New(idem *idempotency.Store) *NotificationHandler {
	return &NotificationHandler{idem: idem}
}

func (h *NotificationHandler) Handle(body []byte) error {
	var event PaymentEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("[Notification] WARN: malformed message, skipping: %v", err)
		return nil
	}

	if h.idem.MarkSeen(event.EventID) {
		log.Printf("[Notification] DUPLICATE event_id=%s – skipping", event.EventID)
		return nil
	}

	dollars := fmt.Sprintf("$%.2f", float64(event.Amount)/100.0)
	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: %s",
		event.CustomerEmail, event.OrderID, dollars)

	return nil
}

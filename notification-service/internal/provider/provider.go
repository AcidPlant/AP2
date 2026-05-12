package provider

import "context"

type Notification struct {
	To      string // recipient email
	OrderID string
	Amount  int64
	Status  string
}

type NotificationProvider interface {
	Send(ctx context.Context, n Notification) error
}

package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository"

	"github.com/google/uuid"
)

func detachedContext() context.Context {
	return context.Background()
}

type PaymentClient interface {
	Authorize(ctx context.Context, orderID string, amount int64) (transactionID string, status string, err error)
}

var (
	ErrNotFound                  = errors.New("order not found")
	ErrCannotCancel              = errors.New("only pending orders can be cancelled")
	ErrInvalidAmount             = errors.New("amount must be greater than 0")
	ErrPaymentServiceUnavailable = errors.New("payment service unavailable")
)

type OrderUseCase interface {
	CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	CancelOrder(ctx context.Context, id string) error
}

type orderUseCase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
}

func NewOrderUseCase(repo repository.OrderRepository, paymentClient PaymentClient) OrderUseCase {
	return &orderUseCase{repo: repo, paymentClient: paymentClient}
}

func (uc *orderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	if idempotencyKey != "" {
		existing, err := uc.repo.FindByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("idempotency lookup: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         domain.StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	_, paymentStatus, err := uc.paymentClient.Authorize(ctx, order.ID, amount)
	if err != nil {
		_ = uc.repo.UpdateStatus(detachedContext(), order.ID, domain.StatusFailed)
		return nil, ErrPaymentServiceUnavailable
	}

	newStatus := domain.StatusPaid
	if paymentStatus == "Declined" {
		newStatus = domain.StatusFailed
	}

	if err := uc.repo.UpdateStatus(ctx, order.ID, newStatus); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	order.Status = newStatus
	return order, nil
}

func (uc *orderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrNotFound
	}
	return order, nil
}

func (uc *orderUseCase) CancelOrder(ctx context.Context, id string) error {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrNotFound
	}
	if order.Status != domain.StatusPending {
		return ErrCannotCancel
	}
	return uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled)
}

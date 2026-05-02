package usecase

import (
	"context"
	"errors"
	"fmt"

	"payment-service/internal/broker"
	"payment-service/internal/domain"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("payment not found")
var ErrInvalidRange = errors.New("min_amount cannot be greater than max_amount")

type PaymentUseCase interface {
	Authorize(ctx context.Context, orderID string, amount int64, customerEmail string) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	ListPayments(ctx context.Context, min, max int64) ([]*domain.Payment, error)
}

type paymentUseCase struct {
	repo      repository.PaymentRepository
	publisher broker.Publisher
}

func NewPaymentUseCase(repo repository.PaymentRepository, pub broker.Publisher) PaymentUseCase {
	return &paymentUseCase{repo: repo, publisher: pub}
}

func (uc *paymentUseCase) Authorize(ctx context.Context, orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	paymentStatus := domain.StatusAuthorized
	if amount > domain.MaxAmount {
		paymentStatus = domain.StatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        paymentStatus,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	if uc.publisher != nil && paymentStatus == domain.StatusAuthorized {
		event := broker.PaymentEvent{
			EventID:       uuid.New().String(),
			OrderID:       orderID,
			Amount:        amount,
			CustomerEmail: customerEmail,
			Status:        paymentStatus,
		}
		if err := uc.publisher.Publish(ctx, event); err != nil {
			fmt.Printf("[WARN] publish payment event: %v\n", err)
		}
	}

	return payment, nil
}

func (uc *paymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p, err := uc.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (uc *paymentUseCase) ListPayments(ctx context.Context, min, max int64) ([]*domain.Payment, error) {
	if min > 0 && max > 0 && min > max {
		return nil, ErrInvalidRange
	}
	return uc.repo.FindByAmountRange(ctx, min, max)
}

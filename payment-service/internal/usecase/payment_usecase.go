package usecase

import (
	"context"
	"errors"
	"fmt"

	"payment-service/internal/domain"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("payment not found")
var ErrInvalidRange = errors.New("min_amount cannot be greater than max_amount")

type PaymentUseCase interface {
	Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	ListPayments(ctx context.Context, min, max int64) ([]*domain.Payment, error)
}

type paymentUseCase struct {
	repo repository.PaymentRepository
}

func NewPaymentUseCase(repo repository.PaymentRepository) PaymentUseCase {
	return &paymentUseCase{repo: repo}
}

func (uc *paymentUseCase) Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	status := domain.StatusAuthorized
	if amount > domain.MaxAmount {
		status = domain.StatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
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

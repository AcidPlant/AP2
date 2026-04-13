package repository

import (
	"context"
	"database/sql"

	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) PaymentRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payments (id, order_id, transaction_id, amount, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.ID, p.OrderID, p.TransactionID, p.Amount, p.Status,
	)
	return err
}

func (r *postgresRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p := &domain.Payment{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, transaction_id, amount, status
		 FROM payments WHERE order_id = $1`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

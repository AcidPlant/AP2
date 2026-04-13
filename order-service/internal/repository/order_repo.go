package repository

import (
	"context"
	"database/sql"

	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status string) error
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
}

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) OrderRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, o *domain.Order) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO orders (id, customer_id, item_name, amount, status, idempotency_key, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		o.ID, o.CustomerID, o.ItemName, o.Amount, o.Status, o.IdempotencyKey, o.CreatedAt,
	)
	return err
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	o := &domain.Order{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.IdempotencyKey, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *postgresRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	if key == "" {
		return nil, nil
	}
	o := &domain.Order{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at
		 FROM orders WHERE idempotency_key = $1`, key,
	).Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.IdempotencyKey, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

package repository

import (
	"context"
	"database/sql"

	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	FindByAmountRange(ctx context.Context, min, max int64) ([]*domain.Payment, error) // NEW
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

func (r *postgresRepo) FindByAmountRange(ctx context.Context, min, max int64) ([]*domain.Payment, error) {
	query := `SELECT id, order_id, transaction_id, amount, status FROM payments WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if min > 0 {
		query += ` AND amount >= $` + itoa(argIdx)
		args = append(args, min)
		argIdx++
	}
	if max > 0 {
		query += ` AND amount <= $` + itoa(argIdx)
		args = append(args, max)
		argIdx++
	}
	query += ` ORDER BY amount ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var payments []*domain.Payment
	for rows.Next() {
		p := &domain.Payment{}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(digits[i%10]) + result
		i /= 10
	}
	return result
}

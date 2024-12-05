package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
)

type dbRepo struct {
	db      *sql.DB
	timeout time.Duration
}

func NewDatabaseRepository(db *sql.DB, timeout time.Duration) order.Repository {
	return &dbRepo{db: db, timeout: timeout}
}

func (d *dbRepo) Add(ctx context.Context, userId string, number string, status order.Status) error {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	_, err := d.db.ExecContext(
		localCtx,
		"INSERT INTO orders (user_id, number, status) VALUES ($1, $2, $3)",
		userId,
		number,
		string(status),
	)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	return nil
}

func (d *dbRepo) Update(ctx context.Context, number string, status order.Status, accrual *int64) error {
	var query string
	var args []any

	if accrual == nil {
		query = "UPDATE orders SET status = $1 WHERE number = $2"
		args = []any{status, number}
	} else {
		query = "UPDATE orders SET status = $1, accrual = $2 WHERE number = $3"
		args = []any{status, *accrual, number}
	}

	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	_, err := d.db.ExecContext(
		localCtx,
		query,
		args...,
	)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	return nil
}

func (d *dbRepo) GetOwner(ctx context.Context, number string) (string, bool, error) {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	row := d.db.QueryRowContext(localCtx, "SELECT user_id FROM orders WHERE number = $1", number)

	var userId string
	err := row.Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("query error: %w", err)
	}

	return userId, true, nil
}

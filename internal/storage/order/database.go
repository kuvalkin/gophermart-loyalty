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

func (d *dbRepo) Add(ctx context.Context, userID string, number string, status order.Status) error {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	_, err := d.db.ExecContext(
		localCtx,
		"INSERT INTO orders (user_id, number, status) VALUES ($1, $2, $3)",
		userID,
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
		query = "UPDATE orders SET status = $1, updated_at = $2 WHERE number = $3"
		args = []any{status, time.Now(), number}
	} else {
		query = "UPDATE orders SET status = $1, updated_at = $2, accrual = $3 WHERE number = $4"
		args = []any{status, time.Now(), *accrual, number}
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

	var userID string
	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("query error: %w", err)
	}

	return userID, true, nil
}

func (d *dbRepo) List(ctx context.Context, userID string) ([]*order.Order, error) {
	rows, err := d.db.QueryContext(
		ctx,
		`SELECT number, status, accrual, updated_at FROM orders WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	defer rows.Close()

	result := make([]*order.Order, 0)
	for rows.Next() {
		var number string
		var status string
		var accrual sql.NullInt64
		var updatedAt time.Time

		if err := rows.Scan(&number, &status, &accrual, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		singleResult := &order.Order{
			Number:     number,
			Status:     order.Status(status),
			UploadedAt: updatedAt,
		}
		if accrual.Valid {
			singleResult.Accrual = &accrual.Int64
		}

		result = append(result, singleResult)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

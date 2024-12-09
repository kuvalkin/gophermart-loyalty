package balance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
)

func NewDatabaseRepository(db *sql.DB, timeout time.Duration) balance.Repository {
	return &dbRepo{db: db, timeout: timeout}
}

type dbRepo struct {
	db      *sql.DB
	timeout time.Duration
}

func (d *dbRepo) Get(ctx context.Context, userID string) (*balance.Balance, bool, error) {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	row := d.db.QueryRowContext(localCtx, "SELECT current, withdrawn FROM balances WHERE user_id = $1", userID)

	b := &balance.Balance{}
	err := row.Scan(&b.Current, &b.Withdrawn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("query error: %w", err)
	}

	return b, true, nil
}

func (d *dbRepo) Increase(ctx context.Context, userID string, increment int64) error {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	_, err := d.db.ExecContext(
		localCtx,
		"INSERT INTO balances (user_id, current, withdrawn) VALUES ($1, $2, 0) ON CONFLICT (user_id) DO UPDATE SET current = current + excluded.current",
		userID,
		increment,
	)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	return nil
}

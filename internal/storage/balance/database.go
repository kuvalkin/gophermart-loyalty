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

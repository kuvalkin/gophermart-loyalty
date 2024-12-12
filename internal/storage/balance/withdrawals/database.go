package withdrawals

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/storage/balance/internal"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func NewDatabaseRepository(db *sql.DB, timeout time.Duration) balance.WithdrawalsRepository {
	return &dbRepo{db: db, timeout: timeout}
}

type dbRepo struct {
	db      *sql.DB
	timeout time.Duration
}

func (d *dbRepo) Add(ctx context.Context, userID string, orderNumber string, sum int64, tx transaction.Transaction) error {
	localCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	_, err := internal.ExecContext(
		localCtx,
		d.db,
		tx,
		"INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)",
		userID,
		orderNumber,
		sum,
	)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	return nil
}

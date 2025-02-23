package internal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func ExecContext(ctx context.Context, db *sql.DB, tx transaction.Transaction, query string, args ...any) (sql.Result, error) {
	var res sql.Result
	var err error
	if tx == nil {
		res, err = db.ExecContext(ctx, query, args...)
	} else {
		dbTx, ok := tx.(*transaction.DatabaseTx)
		if !ok {
			return nil, fmt.Errorf("invalid transaction type: %T", tx)
		}

		res, err = dbTx.DBTx.ExecContext(ctx, query, args...)
	}

	return res, err
}

func QueryRowContext(ctx context.Context, db *sql.DB, tx transaction.Transaction, query string, args ...any) (*sql.Row, error) {
	if tx == nil {
		row := db.QueryRowContext(ctx, query, args...)

		return row, nil
	} else {
		dbTx, ok := tx.(*transaction.DatabaseTx)
		if !ok {
			return nil, fmt.Errorf("invalid transaction type: %T", tx)
		}

		row := dbTx.DBTx.QueryRowContext(ctx, query, args...)

		return row, nil
	}

}

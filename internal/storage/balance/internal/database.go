package internal

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
)

func NewTx(dbTx *sql.Tx) balance.Transaction {
	return &databaseTx{dbTx: dbTx}
}

type databaseTx struct {
	dbTx *sql.Tx
}

func (t *databaseTx) Commit() error {
	return t.dbTx.Commit()
}

func (t *databaseTx) Rollback() error {
	err := t.dbTx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}

	return err
}

func ExecContext(ctx context.Context, db *sql.DB, tx balance.Transaction, query string, args ...any) (sql.Result, error) {
	var res sql.Result
	var err error
	if tx == nil {
		res, err = db.ExecContext(ctx, query, args...)
	} else {
		dbTx := tx.(*databaseTx)

		res, err = dbTx.dbTx.ExecContext(ctx, query, args...)
	}

	return res, err
}

func QueryRowContext(ctx context.Context, db *sql.DB, tx balance.Transaction, query string, args ...any) *sql.Row {
	var row *sql.Row
	if tx == nil {
		row = db.QueryRowContext(ctx, query, args...)
	} else {
		dbTx := tx.(*databaseTx)

		row = dbTx.dbTx.QueryRowContext(ctx, query, args...)
	}

	return row
}

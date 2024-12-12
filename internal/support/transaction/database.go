package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func NewDatabaseTransactionProvider(db *sql.DB) Provider {
	return &databaseProvider{db: db}
}

type databaseProvider struct {
	db *sql.DB
}

func (p *databaseProvider) StartTransaction(ctx context.Context) (Transaction, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cant begin tx: %w", err)
	}

	return newDatabaseTransaction(tx), nil
}

func newDatabaseTransaction(dbTx *sql.Tx) Transaction {
	return &DatabaseTx{DBTx: dbTx}
}

type DatabaseTx struct {
	DBTx *sql.Tx
}

func (t *DatabaseTx) Commit() error {
	return t.DBTx.Commit()
}

func (t *DatabaseTx) Rollback() error {
	err := t.DBTx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}

	return err
}

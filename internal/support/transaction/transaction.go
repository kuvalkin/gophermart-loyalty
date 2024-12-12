package transaction

import "context"

type Provider interface {
	StartTransaction(ctx context.Context) (Transaction, error)
}

type Transaction interface {
	Commit() error
	// Rollback a transaction. If the transaction is already commited or rolled back, should return nil
	Rollback() error
}

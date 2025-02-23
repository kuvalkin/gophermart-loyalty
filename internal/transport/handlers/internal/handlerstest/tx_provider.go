package handlerstest

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func newDummyTxProvider() transaction.Provider {
	return &dummyTxProvider{}
}

type dummyTxProvider struct{}

func (d *dummyTxProvider) StartTransaction(_ context.Context) (transaction.Transaction, error) {
	return &dummyTx{}, nil
}

type dummyTx struct{}

func (d *dummyTx) Commit() error {
	return nil
}

func (d *dummyTx) Rollback() error {
	return nil
}

package balance

import (
	"context"
	"errors"
	"io"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

type Balance struct {
	Current   int64
	Withdrawn int64
}

type WithdrawalHistoryEntry struct {
	OrderNumber string
	Sum         int64
}

var ErrInternal = errors.New("internal error")
var ErrNotEnoughBalance = errors.New("not enough balance")
var ErrInvalidOrderNumber = errors.New("invalid order number")
var ErrInvalidWithdrawalSum = errors.New("invalid withdrawal sum")

type Service interface {
	io.Closer
	Get(ctx context.Context, userID string) (*Balance, error)
	Withdraw(ctx context.Context, userID string, orderNumber string, sum int64) error
	WithdrawalHistory(ctx context.Context, userID string) ([]*WithdrawalHistoryEntry, error)
}

type Repository interface {
	Get(ctx context.Context, userID string, tx transaction.Transaction) (*Balance, bool, error)
	Increase(ctx context.Context, userID string, increment int64) error
	Withdraw(ctx context.Context, userID string, decrement int64, tx transaction.Transaction) error
}

type WithdrawalsRepository interface {
	Add(ctx context.Context, userID string, orderNumber string, sum int64, tx transaction.Transaction) error
}

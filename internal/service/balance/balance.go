package balance

import (
	"context"
	"errors"
	"io"
)

type Balance struct {
	Current   int64
	Withdrawn int64
}

type WithdrawalHistoryEntry struct {
	OrderNumber string
}

var ErrInternal = errors.New("internal error")
var ErrNotEnoughBalance = errors.New("not enough balance")

type Service interface {
	io.Closer
	Get(ctx context.Context, userID string) (*Balance, error)
	Withdraw(ctx context.Context, userID string, orderNumber string, sum int64) error
	WithdrawalHistory(ctx context.Context, userID string) ([]*WithdrawalHistoryEntry, error)
}

type Repository interface {
	Get(ctx context.Context, userID string) (*Balance, bool, error)
	Increase(ctx context.Context, userID string, increment int64) error
}

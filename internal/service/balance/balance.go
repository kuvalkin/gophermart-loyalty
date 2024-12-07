package balance

import (
	"context"
	"errors"
)

type Balance struct {
	Current   int64
	Withdrawn int64
}

type WithdrawalHistoryEntry struct {
	OrderNumber int64
}

var ErrInternal = errors.New("internal error")
var ErrNotEnoughBalance = errors.New("not enough balance")

type Service interface {
	Get(ctx context.Context, userID string) (*Balance, error)
	Increase(ctx context.Context, userID string, orderNumber int64, sum int64) error
	Withdraw(ctx context.Context, userID string, orderNumber int64, sum int64) error
	WithdrawalHistory(ctx context.Context, userID string) ([]*WithdrawalHistoryEntry, error)
}

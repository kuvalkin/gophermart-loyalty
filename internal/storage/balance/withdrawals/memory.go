package withdrawals

import (
	"context"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func NewMemoryRepository() balance.WithdrawalsRepository {
	return &memoryRepo{
		storage: make(map[string][]*balance.WithdrawalHistoryEntry),
	}
}

type memoryRepo struct {
	storage map[string][]*balance.WithdrawalHistoryEntry
}

func (d *memoryRepo) Add(_ context.Context, userID string, orderNumber string, sum int64, _ transaction.Transaction) error {
	_, ok := d.storage[userID]
	if !ok {
		d.storage[userID] = make([]*balance.WithdrawalHistoryEntry, 0)
	}

	d.storage[userID] = append(d.storage[userID], &balance.WithdrawalHistoryEntry{
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	})

	return nil
}

func (d *memoryRepo) List(_ context.Context, userID string) ([]*balance.WithdrawalHistoryEntry, error) {
	list, ok := d.storage[userID]
	if !ok {
		return make([]*balance.WithdrawalHistoryEntry, 0), nil
	}

	return list, nil
}

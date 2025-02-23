package balance

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func NewInMemoryRepository() balance.Repository {
	return &memoryRepo{
		storage: make(map[string]*value),
	}
}

type memoryRepo struct {
	storage map[string]*value
}

type value struct {
	balance *balance.Balance
}

func (m *memoryRepo) Get(_ context.Context, userID string, _ transaction.Transaction) (*balance.Balance, bool, error) {
	value, ok := m.storage[userID]
	if !ok {
		return nil, false, nil
	}

	return value.balance, true, nil
}

func (m *memoryRepo) Increase(_ context.Context, userID string, increment int64) error {
	v, ok := m.storage[userID]
	if !ok {
		v = &value{
			balance: &balance.Balance{},
		}

		m.storage[userID] = v
	}

	v.balance.Current += increment

	return nil
}

func (m *memoryRepo) Withdraw(_ context.Context, userID string, decrement int64, _ transaction.Transaction) error {
	v, ok := m.storage[userID]
	if !ok {
		return nil
	}

	v.balance.Current -= decrement
	v.balance.Withdrawn += decrement

	return nil
}

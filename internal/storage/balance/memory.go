package balance

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
)

func NewInMemoryRepository() balance.Repository {
	return &memoryRepo{
		storage: make(map[string]*value),
	}
}

type memoryRepo struct {
	storage map[string]*value
}

func (m *memoryRepo) Withdraw(ctx context.Context, userID string, decrement int64, tx balance.Transaction) error {
	//TODO implement me
	panic("implement me")
}

type value struct {
	balance *balance.Balance
}

func (m *memoryRepo) Get(ctx context.Context, userID string, tx balance.Transaction) (*balance.Balance, bool, error) {
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

package withdrawals

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/transaction"
)

func NewMemoryRepository() balance.WithdrawalsRepository {
	return &memoryRepo{}
}

type memoryRepo struct {
	storage map[string][]*value
}

type value struct {
	number string
	sum    int64
}

func (d *memoryRepo) Add(_ context.Context, userID string, orderNumber string, sum int64, _ transaction.Transaction) error {
	s, ok := d.storage[userID]
	if !ok {
		s = make([]*value, 0)
		d.storage[userID] = s
	}

	s = append(s, &value{
		number: orderNumber,
		sum:    sum,
	})

	return nil
}

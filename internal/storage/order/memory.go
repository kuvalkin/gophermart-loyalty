package order

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
)

type memoryRepo struct {
	storage map[string]*value
}

type value struct {
	userId  string
	status  order.Status
	accrual *int64
}

func NewInMemoryRepository() order.Repository {
	return &memoryRepo{
		storage: make(map[string]*value),
	}
}

func (m *memoryRepo) Add(_ context.Context, userId string, number string, status order.Status) error {
	m.storage[number] = &value{
		userId: userId,
		status: status,
	}

	return nil
}

func (m *memoryRepo) Update(_ context.Context, number string, status order.Status, accrual *int64) error {
	value, ok := m.storage[number]
	if !ok {
		return nil
	}

	value.status = status
	if accrual != nil {
		value.accrual = accrual
	}

	return nil
}

func (m *memoryRepo) GetOwner(_ context.Context, number string) (string, bool, error) {
	value, ok := m.storage[number]
	if !ok {
		return "", false, nil
	}

	return value.userId, true, nil
}

package order

import (
	"context"
	"time"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
)

type memoryRepo struct {
	storage map[string]*value
}

type value struct {
	userID     string
	status     order.Status
	accrual    *int64
	uploadedAt time.Time
}

func NewInMemoryRepository() order.Repository {
	return &memoryRepo{
		storage: make(map[string]*value),
	}
}

func (m *memoryRepo) Add(_ context.Context, userID string, number string, status order.Status) error {
	m.storage[number] = &value{
		userID:     userID,
		status:     status,
		uploadedAt: time.Now(),
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

	return value.userID, true, nil
}

func (m *memoryRepo) List(_ context.Context, userID string) ([]*order.Order, error) {
	result := make([]*order.Order, 0)

	for number, value := range m.storage {
		if value.userID != userID {
			continue
		}

		result = append(result, &order.Order{
			Number:     number,
			Status:     value.status,
			Accrual:    value.accrual,
			UploadedAt: value.uploadedAt,
		})
	}

	return result, nil
}

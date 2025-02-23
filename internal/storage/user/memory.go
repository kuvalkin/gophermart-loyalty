package user

import (
	"context"

	"github.com/google/uuid"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
)

type memoryRepo struct {
	storage map[string]*value
}

type value struct {
	id   string
	hash string
}

func NewInMemoryRepository() user.Repository {
	return &memoryRepo{
		storage: make(map[string]*value),
	}
}

func (d *memoryRepo) Add(_ context.Context, login string, passwordHash string) error {
	if _, exists := d.storage[login]; exists {
		return user.ErrLoginNotUnique
	}

	d.storage[login] = &value{
		id:   uuid.New().String(),
		hash: passwordHash,
	}

	return nil
}

func (d *memoryRepo) Find(_ context.Context, login string) (string, string, bool, error) {
	value, ok := d.storage[login]
	if !ok {
		return "", "", false, nil
	}

	return value.id, value.hash, true, nil
}

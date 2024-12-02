package user

import (
	"context"
)

type memoryRepo struct {
	storage map[string]string
}

func NewInMemoryRepository() Repository {
	return &memoryRepo{
		storage: make(map[string]string),
	}
}

func (d *memoryRepo) Add(_ context.Context, login string, passwordHash string) error {
	if _, exists := d.storage[login]; exists {
		return ErrLoginNotUnique
	}

	d.storage[login] = passwordHash

	return nil
}

func (d *memoryRepo) GetPasswordHash(_ context.Context, login string) (string, bool, error) {
	hash, ok := d.storage[login]

	return hash, ok, nil
}

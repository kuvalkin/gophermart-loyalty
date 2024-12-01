package user

import (
	"context"
	"errors"
)

var ErrLoginNotUnique = errors.New("user with this login already exists")

type Repository interface {
	Add(ctx context.Context, login string, passwordHash string) error
	GetPasswordHash(ctx context.Context, login string) (string, bool, error)
}

package user

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidLogin = errors.New("invalid login")
var ErrInvalidPassword = errors.New("password is too short")
var ErrLoginTaken = errors.New("user with this login already exists")
var ErrInvalidPair = errors.New("login/password pair is invalid")
var ErrInvalidToken = errors.New("invalid token")
var ErrInternal = errors.New("internal error")

type Service interface {
	Register(ctx context.Context, login string, password string) error
	// Login authenticates a user and returns auth token on success
	Login(ctx context.Context, login string, password string) (string, error)
	ParseToken(ctx context.Context, token string) (string, error)
}

type Options struct {
	TokenSecret           []byte
	PasswordSalt          string
	MinPasswordLength     int
	TokenExpirationPeriod time.Duration
}

var ErrLoginNotUnique = errors.New("user with this login already exists")

type Repository interface {
	Add(ctx context.Context, login string, passwordHash string) error
	Find(ctx context.Context, login string) (string, string, bool, error)
}

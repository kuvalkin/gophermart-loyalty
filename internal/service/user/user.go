package user

import (
	"context"

	"github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
)

type Service interface {
	Register(ctx context.Context, login string, password string) error
	// Login authenticates a user and returns auth token on success
	Login(ctx context.Context, login string, password string) (string, error)
	CheckToken(ctx context.Context, token string) error
}

func NewService(repo user.Repository, tokenSecret []byte, passwordSalt string) Service {
	return &service{
		repo:         repo,
		tokenSecret:  tokenSecret,
		passwordSalt: passwordSalt,
	}
}

type service struct {
	repo         user.Repository
	tokenSecret  []byte
	passwordSalt string
}

func (s *service) Register(ctx context.Context, login string, password string) error {
	//TODO implement me
	panic("implement me")
}

func (s *service) Login(ctx context.Context, login string, password string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *service) CheckToken(ctx context.Context, token string) error {
	//TODO implement me
	panic("implement me")
}

package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
)

type Service interface {
	Register(ctx context.Context, login string, password string) error
	// Login authenticates a user and returns auth token on success
	Login(ctx context.Context, login string, password string) (string, error)
	CheckToken(ctx context.Context, token string) error
}

type Options struct {
	TokenSecret           []byte
	PasswordSalt          string
	MinPasswordLength     int
	TokenExpirationPeriod time.Duration
}

var signingMethod = jwt.SigningMethodHS256

func NewService(repo user.Repository, options *Options) (Service, error) {
	if options == nil {
		return nil, errors.New("no options provided")
	}

	return &service{
		repo:    repo,
		options: options,
		logger:  log.Logger().Named("userService"),
	}, nil
}

type service struct {
	repo    user.Repository
	options *Options
	logger  *zap.SugaredLogger
}

func (s *service) Register(ctx context.Context, login string, password string) error {
	if len(password) < s.options.MinPasswordLength {
		return fmt.Errorf("password length must be at least %d characters", s.options.MinPasswordLength)
	}

	hash := s.hashPassword(password)

	err := s.repo.Add(ctx, login, hash)
	if err != nil {
		if errors.Is(err, user.ErrLoginNotUnique) {
			// do not say explicitly that login is taken in case of bruteforce
			return fmt.Errorf("login is invalid")
		}

		s.logger.Errorw("user adding failed", "error", err)

		return errors.New("failed to register user")
	}

	return nil
}

func (s *service) Login(ctx context.Context, login string, password string) (string, error) {
	savedHash, found, err := s.repo.GetPasswordHash(ctx, login)
	if err != nil {
		s.logger.Errorw("failed to fetch password hash", "login", login, "error", err)

		return "", errors.New("storage read error")
	}

	invalidPairErr := errors.New("invalid login/password pair")

	if !found {
		return "", invalidPairErr
	}

	if savedHash != s.hashPassword(password) {
		return "", invalidPairErr
	}

	token, err := s.issueToken()
	if err != nil {
		s.logger.Errorw("failed to issue token", "login", login, "error", err)

		return "", errors.New("failed to issue token")
	}

	return token, nil
}

func (s *service) CheckToken(_ context.Context, token string) error {
	parsedToken, err := jwt.Parse(
		token,
		func(t *jwt.Token) (interface{}, error) {
			method, ok := t.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}

			if method.Name != signingMethod.Name {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}

			return s.options.TokenSecret, nil
		},
	)

	if err != nil {
		s.logger.Infow("failed to parse token", "error", err)

		return fmt.Errorf("invalid token")
	}

	if !parsedToken.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func (s *service) hashPassword(password string) string {
	withSalt := password + s.options.PasswordSalt

	hashBytes := sha256.Sum256([]byte(withSalt))

	return hex.EncodeToString(hashBytes[:])
}

func (s *service) issueToken() (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(signingMethod, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.options.TokenExpirationPeriod)),
	})

	tokenString, err := token.SignedString(s.options.TokenSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

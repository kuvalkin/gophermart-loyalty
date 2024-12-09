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

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

var signingMethod = jwt.SigningMethodHS256

func NewService(repo Repository, options *Options) (Service, error) {
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
	repo    Repository
	options *Options
	logger  *zap.SugaredLogger
}

func (s *service) Register(ctx context.Context, login string, password string) error {
	if login == "" {
		return ErrInvalidLogin
	}

	if len(password) < s.options.MinPasswordLength {
		return ErrInvalidPassword
	}

	hash := s.hashPassword(password)

	err := s.repo.Add(ctx, login, hash)
	if err != nil {
		if errors.Is(err, ErrLoginNotUnique) {
			return ErrLoginTaken
		}

		s.logger.Errorw("user adding failed", "error", err)

		return ErrInternal
	}

	return nil
}

func (s *service) Login(ctx context.Context, login string, password string) (string, error) {
	id, savedHash, found, err := s.repo.Find(ctx, login)
	if err != nil {
		s.logger.Errorw("failed to fetch password hash", "login", login, "error", err)

		return "", ErrInternal
	}

	if !found {
		return "", ErrInvalidPair
	}

	if savedHash != s.hashPassword(password) {
		return "", ErrInvalidPair
	}

	token, err := s.issueToken(id)
	if err != nil {
		s.logger.Errorw("failed to issue token", "login", login, "error", err)

		return "", ErrInternal
	}

	return token, nil
}

func (s *service) ParseToken(_ context.Context, token string) (string, error) {
	claims := new(jwt.RegisteredClaims)

	parsedToken, err := jwt.ParseWithClaims(
		token,
		claims,
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

		return "", ErrInvalidToken
	}

	if !parsedToken.Valid {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}

func (s *service) hashPassword(password string) string {
	withSalt := password + s.options.PasswordSalt

	hashBytes := sha256.Sum256([]byte(withSalt))

	return hex.EncodeToString(hashBytes[:])
}

func (s *service) issueToken(userID string) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(signingMethod, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.options.TokenExpirationPeriod)),
	})

	tokenString, err := token.SignedString(s.options.TokenSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

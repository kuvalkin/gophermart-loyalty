package balance

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

func NewService(repo Repository) (Service, error) {
	s := &service{
		repo:   repo,
		logger: log.Logger().Named("balanceService"),
	}

	err := event.Subscribe("order:processed", s.onOrderProcessed)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to order processed event: %w", err)
	}

	return s, nil
}

type service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func (s *service) Get(ctx context.Context, userID string) (*Balance, error) {
	b, found, err := s.repo.Get(ctx, userID)
	if err != nil {
		s.logger.Errorw("error getting balance", "userID", userID, "error", err)

		return nil, ErrInternal
	}
	if !found {
		s.logger.Debugw("balance not found, returning empty value", "userID", userID)

		// there is no record in repo until user withdraws or uploads smth
		return &Balance{}, nil
	}

	return b, nil
}

func (s *service) Withdraw(ctx context.Context, userID string, orderNumber string, sum int64) error {
	//TODO implement me
	panic("implement me")
}

func (s *service) WithdrawalHistory(ctx context.Context, userID string) ([]*WithdrawalHistoryEntry, error) {
	//TODO implement me
	panic("implement me")
}

func (s *service) Close() error {
	err := event.Unsubscribe("order:processed", s.onOrderProcessed)

	if err != nil {
		return fmt.Errorf("failed to unsubscribe to order processed event: %w", err)
	}

	return nil
}

func (s *service) onOrderProcessed(userID string, accrual int64) {
	err := s.repo.Increase(context.Background(), userID, accrual)
	if err != nil {
		s.logger.Errorw("failed to increase balance", "userID", userID, "error", err)
	}
}

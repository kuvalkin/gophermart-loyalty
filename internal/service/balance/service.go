package balance

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/order"
)

func NewService(repo Repository, wRepo WithdrawalsRepository, txProvider TransactionProvider) (Service, error) {
	s := &service{
		repo:            repo,
		withdrawalsRepo: wRepo,
		txProvider:      txProvider,
		logger:          log.Logger().Named("balanceService"),
	}

	err := event.Subscribe("order:processed", s.onOrderProcessed)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to order processed event: %w", err)
	}

	return s, nil
}

type service struct {
	repo            Repository
	withdrawalsRepo WithdrawalsRepository
	txProvider      TransactionProvider
	logger          *zap.SugaredLogger
}

func (s *service) Get(ctx context.Context, userID string) (*Balance, error) {
	b, found, err := s.repo.Get(ctx, userID, nil)
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
	localLogger := s.logger.WithLazy("userID", userID, "orderNumber", orderNumber, "sum", sum)

	err := order.ValidateNumber(orderNumber)
	if err != nil {
		localLogger.Debugw("invalid order number", "error", err)

		return ErrInvalidOrderNumber
	}

	if sum <= 0 {
		localLogger.Debugw("invalid sum")

		return ErrInvalidWithdrawalSum
	}

	tx, err := s.txProvider.StartTransaction(ctx)
	if err != nil {
		localLogger.Errorw("error starting transaction", "error", err)

		return ErrInternal
	}

	defer func() {
		err := tx.Rollback()
		if err != nil {
			localLogger.Errorw("error rolling back transaction", "error", err)
		}
	}()

	b, found, err := s.repo.Get(ctx, userID, tx)
	if err != nil {
		localLogger.Errorw("error getting balance", "error", err)

		return ErrInternal
	}

	if !found {
		localLogger.Debugw("balance not found")

		return ErrNotEnoughBalance
	}

	if sum < b.Current {
		localLogger.Debugw("balance is insufficient", "current", b.Current)

		return ErrNotEnoughBalance
	}

	err = s.repo.Withdraw(ctx, userID, sum, tx)
	if err != nil {
		localLogger.Errorw("error withdrawing balance", "error", err)

		return ErrInternal
	}

	err = s.withdrawalsRepo.Add(ctx, userID, orderNumber, sum, tx)
	if err != nil {
		localLogger.Errorw("error writing history", "error", err)

		return ErrInternal
	}

	err = tx.Commit()
	if err != nil {
		localLogger.Errorw("error committing transaction", "error", err)

		return ErrInternal
	}

	return nil
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

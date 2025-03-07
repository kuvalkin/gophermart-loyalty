package order

import (
	"context"

	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/order"
)

func NewService(repo Repository, poller AccrualPoller) Service {
	return &service{
		repo:   repo,
		poller: poller,
		logger: log.Logger().Named("orderService"),
	}
}

type service struct {
	repo   Repository
	poller AccrualPoller
	logger *zap.SugaredLogger
}

func (s *service) Upload(ctx context.Context, userID string, number string) error {
	localLogger := s.logger.WithLazy("userID", userID, "number", number)

	err := order.ValidateNumber(number)
	if err != nil {
		localLogger.Debugw("invalid number", "error", err)

		return ErrInvalidNumber
	}

	ownerID, found, err := s.repo.GetOwner(ctx, number)
	if err != nil {
		localLogger.Errorw("can't get owner", "error", err)

		return ErrInternal
	}

	if found {
		if ownerID == userID {
			return ErrAlreadyUploaded
		} else {
			return ErrUploadedByAnotherUser
		}
	}

	err = s.repo.Add(ctx, userID, number, StatusNew)
	if err != nil {
		localLogger.Errorw("can't add new order", "error", err)

		return ErrInternal
	}

	err = s.AddToProcessQueue(number, userID, StatusNew)
	if err != nil {
		localLogger.Errorw("can't add to process queue", "error", err)

		return ErrInternal
	}

	return nil
}

func (s *service) AddToProcessQueue(number, userID string, currentStatus Status) error {
	if currentStatus.IsFinal() {
		return ErrAlreadyProcessed
	}

	resultChan, err := s.poller.Enqueue(number, StatusNew)
	if err != nil {
		s.logger.Errorw("can't add to queue", "error", err, "number", number, "currentStatus", currentStatus)

		return ErrInternal
	}

	go s.listenAccrualResults(resultChan, number, userID)

	return nil
}

func (s *service) listenAccrualResults(resultChan <-chan AccrualResult, number string, userID string) {
	localLogger := s.logger.WithLazy("number", number)

	for result := range resultChan {
		newStatus := result.Status

		if result.Err != nil {
			localLogger.Errorw("received error from accrual queue, marking order as invalid", "error", result.Err)

			newStatus = StatusInvalid
		}

		err := s.repo.Update(
			context.Background(),
			number,
			newStatus,
			result.Accrual,
		)

		if err != nil {
			localLogger.Errorw("can't update order", "error", err)
			continue
		}

		if result.Status == StatusProcessed && result.Accrual != nil {
			event.Publish("order:processed", userID, *result.Accrual)
		}
	}
}

func (s *service) List(ctx context.Context, userID string) ([]*Order, error) {
	list, err := s.repo.List(ctx, userID)
	if err != nil {
		s.logger.Errorw("can't list orders", "userID", userID, "error", err)

		return nil, ErrInternal
	}

	return list, nil
}

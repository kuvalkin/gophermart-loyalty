package order

import (
	"context"
	"errors"

	"github.com/ShiraazMoollatjie/goluhn"
	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
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

func (s *service) Upload(ctx context.Context, userId string, number string) error {
	localLogger := s.logger.WithLazy("userId", userId, "number", number)

	err := checkNumber(number)
	if err != nil {
		localLogger.Debugw("invalid number", "error", err)

		return ErrInvalidNumber
	}

	ownerId, found, err := s.repo.GetOwner(ctx, number)
	if err != nil {
		localLogger.Errorw("can't get owner", "error", err)

		return ErrInternal
	}

	if found {
		if ownerId == userId {
			return ErrAlreadyUploaded
		} else {
			return ErrUploadedByAnotherUser
		}
	}

	err = s.repo.Add(ctx, userId, number, StatusNew)
	if err != nil {
		localLogger.Errorw("can't add new order", "error", err)

		return ErrInternal
	}

	err = s.AddToProcessQueue(number, StatusNew)
	if err != nil {
		localLogger.Errorw("can't add to process queue", "error", err)

		return ErrInternal
	}

	return nil
}

func (s *service) AddToProcessQueue(number string, currentStatus Status) error {
	if currentStatus.IsFinal() {
		return ErrAlreadyProcessed
	}

	resultChan, err := s.poller.Enqueue(number, StatusNew)
	if err != nil {
		s.logger.Errorw("can't add to queue", "error", err, "number", number, "currentStatus", currentStatus)

		return ErrInternal
	}

	go s.listenAccrualResults(resultChan, number)

	return nil
}

func (s *service) listenAccrualResults(resultChan <-chan AccrualResult, number string) {
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
			event.Publish("order:processed", *result.Accrual)
		}
	}
}

func (s *service) List(ctx context.Context, userId string) ([]*Order, error) {
	//TODO implement me
	panic("implement me")
}

func checkNumber(number string) error {
	if number == "" {
		// goluhn doesnt return error on empty string
		return errors.New("empty number")
	}

	return goluhn.Validate(number)
}

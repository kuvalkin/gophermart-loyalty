package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/ShiraazMoollatjie/goluhn"
	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/event"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

func NewService(repo Repository, options *Options) (Service, error) {
	if options == nil {
		return nil, errors.New("no options provided")
	}

	a, err := newAccrual(options.AccrualSystemAddress, options.MaxRetriesToAccrual, options.MaxAccrualRetryWaitTime)
	if err != nil {
		return nil, fmt.Errorf("cant create accrual: %w", err)
	}

	return &service{
		options: options,
		repo:    repo,
		accrual: a,
		logger:  log.Logger().Named("orderService"),
	}, nil
}

type service struct {
	options *Options
	repo    Repository
	accrual *accrual
	logger  *zap.SugaredLogger
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

	resultChan, err := s.accrual.addToQueue(number, StatusNew)
	if err != nil {
		localLogger.Errorw("can't add to queue", "error", err)

		return ErrInternal
	}

	go s.listenAccrualResults(resultChan, number)

	return nil
}

func (s *service) listenAccrualResults(resultChan <-chan accrualResult, number string) {
	localLogger := s.logger.WithLazy("number", number)

	for result := range resultChan {
		newStatus := result.status

		if result.err != nil {
			localLogger.Errorw("received error from accrual queue, marking order as invalid", "error", result.err)

			newStatus = StatusInvalid
		}

		err := s.repo.Update(
			context.Background(),
			number,
			newStatus,
			result.accrual,
		)

		if err != nil {
			localLogger.Errorw("can't update order", "error", err)
			continue
		}

		if result.status == StatusProcessed && result.accrual != nil {
			event.Publish("order:processed", *result.accrual)
		}
	}
}

func (s *service) List(ctx context.Context, userId string) ([]*Order, error) {
	//TODO implement me
	panic("implement me")
}

func checkNumber(number string) error {
	return goluhn.Validate(number)
}

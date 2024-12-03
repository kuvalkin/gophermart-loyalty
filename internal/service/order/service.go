package order

import (
	"context"
	"errors"

	"github.com/ShiraazMoollatjie/goluhn"
	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/log"
)

func NewService(repo Repository, options *Options) (Service, error) {
	if options == nil {
		return nil, errors.New("no options provided")
	}

	return &service{
		options: options,
		repo:    repo,
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
	wrappedLogger := s.logger.WithLazy("userId", userId, "number", number)

	err := checkNumber(number)
	if err != nil {
		wrappedLogger.Debugw("invalid number", "error", err)

		return ErrInvalidNumber
	}

	ownerId, found, err := s.repo.GetOwner(ctx, number)
	if err != nil {
		wrappedLogger.Errorw("can't get owner", "error", err)

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
		wrappedLogger.Errorw("can't add new order", "error", err)

		return ErrInternal
	}

	resultChan, err := s.accrual.addToQueue(number)
	if err != nil {
		wrappedLogger.Errorw("can't add to queue", "error", err)

		return ErrInternal
	}

	go func() {
		result := <-resultChan
		//update status
		// emit event
	}()

	return nil
}

func (s *service) List(ctx context.Context, userId string) ([]*Order, error) {
	//TODO implement me
	panic("implement me")
}

func checkNumber(number string) error {
	return goluhn.Validate(number)
}

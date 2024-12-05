package order

import (
	"context"
	"errors"
	"time"
)

type Status string

const StatusNew = Status("NEW")
const StatusProcessing = Status("PROCESSING")
const StatusInvalid = Status("INVALID")
const StatusProcessed = Status("PROCESSED")

type Order struct {
	Number     string
	Status     Status
	Accrual    *int64
	UploadedAt time.Time
}

var ErrAlreadyUploaded = errors.New("order already uploaded")
var ErrUploadedByAnotherUser = errors.New("uploaded by another user")
var ErrInvalidNumber = errors.New("invalid order number")
var ErrInternal = errors.New("internal error")

type Service interface {
	Upload(ctx context.Context, userId string, number string) error
	List(ctx context.Context, userId string) ([]*Order, error)
}

type Repository interface {
	Add(ctx context.Context, userId string, number string, status Status) error
	Update(ctx context.Context, number string, status Status, accrual *int64) error
	GetOwner(ctx context.Context, number string) (string, bool, error)
}

type Options struct {
	AccrualSystemAddress    string
	MaxRetriesToAccrual     int
	MaxAccrualRetryWaitTime time.Duration
}

package order

import (
	"fmt"

	"golang.org/x/time/rate"

	"github.com/kuvalkin/gophermart-loyalty/internal/pool"
)

type accrual struct {
	pool       *pool.Pool
	limiter    *rate.Limiter
	maxRetries int
}

type accrualResult struct {
	amount int64
	err    error
}

type accrualTask struct {
	number  string
	retries int
}

func newAccrual(maxRetries int) (*accrual, error) {
	// todo call Release on shutdown?
	p, err := pool.NewPool(nil)
	if err != nil {
		return nil, fmt.Errorf("cant create a new pool: %w", err)
	}

	return &accrual{
		pool:       p,
		limiter:    rate.NewLimiter(rate.Inf, 1),
		maxRetries: maxRetries,
	}, nil
}

func (a accrual) addToQueue(number string) (<-chan accrualResult, error) {
	result := make(chan accrualResult)

	err := a.pool.Submit(func() {
		// make request

		result <- accrualResult{}
		close(result)
	})
	if err != nil {
		return nil, fmt.Errorf("cant submit to task to pool: %w", err)
	}

	return result, nil
}

package pool

import (
	"fmt"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

type Pool struct {
	pool *ants.Pool
}

func NewPool(workers *int) (*Pool, error) {
	var wInt int
	if workers != nil {
		wInt = *workers
	} else {
		wInt = ants.DefaultAntsPoolSize
	}

	poolLogger := log.Logger().Named("pool")

	pool, err := ants.NewPool(
		wInt,
		ants.WithLogger(&antsLogger{logger: poolLogger}),
	)
	if err != nil {
		return nil, fmt.Errorf("cant create pool: %w", err)
	}

	return &Pool{
		pool: pool,
	}, nil
}

func (p *Pool) Release() {
	p.pool.Release()
}

func (p *Pool) Tune(newMaxWorkers int) {
	p.pool.Tune(newMaxWorkers)
}

func (p *Pool) Submit(task func()) error {
	return p.pool.Submit(task)
}

type antsLogger struct {
	logger *zap.SugaredLogger
}

func (a *antsLogger) Printf(format string, args ...interface{}) {
	a.logger.Debugf(format, args...)
}

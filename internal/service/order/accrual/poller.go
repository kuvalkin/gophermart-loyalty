package accrual

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/pool"
)

type poller struct {
	pool             *pool.Pool
	limiter          *rate.Limiter
	tuningMutex      *sync.Mutex
	client           *resty.Client
	timeout          time.Duration
	maxAttempts      int
	maxRetryWaitTime time.Duration
	taskList         *taskList
	logger           *zap.SugaredLogger
}

type task struct {
	number      string
	knownStatus order.Status
	attempts    int
	resultChan  chan<- order.AccrualResult
}

type taskList struct {
	sync.Mutex
	tasks map[string]*task
}

func (t *taskList) deleteSingle(number string) {
	t.Lock()
	defer t.Unlock()

	delete(t.tasks, number)
}

func NewPoller(accrualSystemAddress string, timeout time.Duration, maxRetries int, maxRetryWaitTime time.Duration) (order.AccrualPoller, error) {
	p, err := pool.NewPool(nil)
	if err != nil {
		return nil, fmt.Errorf("cant create a new pool: %w", err)
	}

	return &poller{
		pool:             p,
		limiter:          rate.NewLimiter(rate.Inf, 1), // no limit by default
		tuningMutex:      &sync.Mutex{},
		client:           resty.New().SetTimeout(timeout).SetBaseURL(accrualSystemAddress),
		timeout:          timeout,
		maxAttempts:      maxRetries,
		maxRetryWaitTime: maxRetryWaitTime,
		taskList: &taskList{
			tasks: make(map[string]*task),
		},
		logger: log.Logger().Named("accrualPoller"),
	}, nil
}

func (p *poller) Enqueue(number string, currentStatus order.Status) (<-chan order.AccrualResult, error) {
	p.taskList.Lock()
	defer p.taskList.Unlock()

	if _, exists := p.taskList.tasks[number]; exists {
		return nil, fmt.Errorf("order %s already enqueued", number)
	}

	result := make(chan order.AccrualResult, 1)

	task := &task{
		number:      number,
		knownStatus: currentStatus,
		resultChan:  result,
		attempts:    0,
	}

	// enqueue may take a while if all workers are busy
	err := p.enqueue(task)
	if err != nil {
		return nil, fmt.Errorf("cant submit to task to pool: %w", err)
	}

	p.taskList.tasks[number] = task

	return result, nil
}

func (p *poller) Close() error {
	err := p.pool.Close()
	if err != nil {
		return fmt.Errorf("cant close the pool: %w", err)
	}

	p.taskList.Lock()
	defer p.taskList.Unlock()

	for _, task := range p.taskList.tasks {
		close(task.resultChan)
	}

	return nil
}

func (p *poller) enqueue(task *task) error {
	return p.pool.Submit(func() {
		p.processTask(task)
	})
}

func (p *poller) processTask(task *task) {
	// we need an actual deadline for wait, because if we start waiting while requests are blocked, we will wait forever
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	p.logger.Debugw("waiting to make a request", "number", task.number)
	err := p.limiter.Wait(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "would exceed context deadline") || ctx.Err() != nil {
			p.logger.Debugw("too much to wait to make a request, try this task later", "number", task.number, "error", err)
		} else {
			p.logger.Errorw("rate limiter wait error", "number", task.number, "error", err)
		}

		p.retryLaterOrCloseTask(task)
		return
	}

	// task attempts should not be largely affected by rate limiting
	task.attempts++

	receivedStatus, amount, err := p.makeRequest(task.number)

	var isCompleted bool
	if err == nil {
		isCompleted = p.notifyAboutChanges(task, receivedStatus, amount)
	} else {
		p.logger.Errorw("error making request to accrual service", "number", task.number, "error", err)
		isCompleted = false
	}

	if isCompleted {
		close(task.resultChan)
		p.taskList.deleteSingle(task.number)
	} else {
		p.retryLaterOrCloseTask(task)
	}
}

func (p *poller) makeRequest(number string) (accrualStatus, int64, error) {
	wrapped := p.logger.WithLazy("number", number)
	payload := new(accrualResponse)

	wrapped.Debug("making request")
	response, err := p.client.R().
		SetPathParam("number", number).
		SetResult(payload).
		Get("/api/orders/{number}")

	if err != nil {
		return "", 0, fmt.Errorf("cant make p request: %w", err)
	}

	wrapped.Debug("got response")

	if response.StatusCode() == http.StatusTooManyRequests {
		p.tuneRateLimiting(response)

		return "", 0, fmt.Errorf("rate limited")
	}

	if response.StatusCode() == http.StatusNoContent {
		return "", 0, nil
	}

	if response.StatusCode() != http.StatusOK {
		return "", 0, fmt.Errorf("unexpected status code: %d", response.StatusCode())
	}

	s, err := statusFromString(payload.Status)
	if err != nil {
		return "", 0, fmt.Errorf("cant parse status from response: %w", err)
	}

	return s, int64(payload.Accrual * 100), nil
}

func (p *poller) tuneRateLimiting(response *resty.Response) {
	p.tuningMutex.Lock()
	defer p.tuningMutex.Unlock()

	retryAfter, numberOfRequests, period := p.parseRateLimitedResponse(response)
	p.logger.Infow("tuning rate limiting", "retryAfter", retryAfter, "numberOfRequests", numberOfRequests, "period", period)

	// block new requests until rate limitation has passed
	// https://github.com/golang/go/issues/18763
	p.limiter.SetLimit(math.SmallestNonzeroFloat64)
	p.logger.Infow("all new requests are blocked temporary", "whenSeconds", retryAfter)

	// tune limit according to response
	time.AfterFunc(time.Duration(retryAfter)*time.Second, func() {
		p.logger.Info("new rate limited installed")
		p.limiter.SetLimit(rate.Limit(numberOfRequests / period))
	})

	// limit number of goroutines
	p.pool.Tune(numberOfRequests)
}

func (p *poller) parseRateLimitedResponse(response *resty.Response) (int, int, int) {
	retryAfterHeader := response.Header().Get("Retry-After")
	retryAfterSeconds, err := strconv.Atoi(retryAfterHeader)
	if err != nil {
		p.logger.Warnw("cant parse Retry-After header, using default value", "header", retryAfterHeader)

		retryAfterSeconds = 60
	}

	defaultNumberOfRequests := 1
	defaultPeriod := 60

	body := strings.TrimSpace(response.String())
	matches := regexp.MustCompile(`^No more than (\d+) requests per (second|minute|hour) allowed$`).FindStringSubmatch(body)
	if len(matches) != 3 {
		p.logger.Errorw("cant parse rate limiting response, using default values", "body", body, "matches", matches)

		return retryAfterSeconds, defaultNumberOfRequests, defaultPeriod
	}

	numberOfRequests, err := strconv.Atoi(matches[1])
	if err != nil {
		p.logger.Errorw("cant parse number of requests, using default value", "number", matches[1])

		numberOfRequests = defaultNumberOfRequests
	}

	period := defaultPeriod

	switch matches[2] {
	case "second":
		period = 1
	case "minute":
		period = 60
	case "hour":
		period = 60 * 60
	default:
		p.logger.Errorw("cant parse rate limiting period, using default values", "period", matches[2])
	}

	return numberOfRequests, period, numberOfRequests
}

func (p *poller) retryLaterOrCloseTask(task *task) {
	err := p.maybeRetryLater(task)

	if err != nil {
		task.resultChan <- order.AccrualResult{
			Err: err,
		}

		close(task.resultChan)
		p.taskList.deleteSingle(task.number)
	}
}

func (p *poller) maybeRetryLater(task *task) error {
	if task.attempts > p.maxAttempts {
		return errors.New("max attempts exceeded")
	}

	after := p.calcRetryPeriod(task.attempts)

	time.AfterFunc(after, func() {
		err := p.enqueue(task)
		if err != nil {
			p.logger.Errorw("cant submit to task to pool", "number", task.number, "error", err)
		}
	})

	return nil
}

func (p *poller) calcRetryPeriod(attempt int) time.Duration {
	// capped exponential backoff
	interval := math.Min(float64(p.maxRetryWaitTime), float64(time.Second)*math.Exp2(float64(attempt)))

	return time.Duration(interval)
}

func (p *poller) notifyAboutChanges(task *task, receivedStatus accrualStatus, receivedAccrual int64) bool {
	orderStatus := receivedStatus.orderStatus()

	if orderStatus == order.StatusProcessed {
		task.resultChan <- order.AccrualResult{
			Status:  orderStatus,
			Accrual: &receivedAccrual,
		}

		return true
	}

	if orderStatus.IsFinal() {
		task.resultChan <- order.AccrualResult{
			Status: orderStatus,
		}

		return true
	}

	if orderStatus != task.knownStatus {
		task.knownStatus = orderStatus
		task.resultChan <- order.AccrualResult{
			Status: orderStatus,
		}
	}

	return false
}

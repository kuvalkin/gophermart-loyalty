package accrual

import (
	"context"
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
	maxAttempts      int
	maxRetryWaitTime time.Duration
	logger           *zap.SugaredLogger
}

type task struct {
	number      string
	knownStatus order.Status
	attempts    int
	resultChan  chan<- order.AccrualResult
}

func NewPoller(accrualSystemAddress string, maxRetries int, maxRetryWaitTime time.Duration) (order.AccrualPoller, error) {
	// todo call Release on shutdown?
	p, err := pool.NewPool(nil)
	if err != nil {
		return nil, fmt.Errorf("cant create a new pool: %w", err)
	}

	return &poller{
		pool:             p,
		limiter:          rate.NewLimiter(rate.Inf, 1),
		tuningMutex:      &sync.Mutex{},
		client:           resty.New().SetBaseURL(accrualSystemAddress),
		maxAttempts:      maxRetries,
		maxRetryWaitTime: maxRetryWaitTime,
		logger:           log.Logger().Named("accrualPoller"),
	}, nil
}

func (p *poller) Enqueue(number string, currentStatus order.Status) (<-chan order.AccrualResult, error) {
	result := make(chan order.AccrualResult, 1)

	task := &task{
		number:      number,
		knownStatus: currentStatus,
		resultChan:  result,
		attempts:    0,
	}

	err := p.enqueue(task)
	if err != nil {
		return nil, fmt.Errorf("cant submit to task to pool: %w", err)
	}

	return result, nil
}

func (p *poller) enqueue(task *task) error {
	return p.pool.Submit(func() {
		p.processTask(task)
	})
}

func (p *poller) processTask(task *task) {
	task.attempts++

	status, amount, err := p.makeRequest(task.number)
	if err != nil {
		if task.attempts > p.maxAttempts {
			task.resultChan <- order.AccrualResult{
				Err: fmt.Errorf("max attempts exceeded"),
			}

			close(task.resultChan)

			return
		}

		p.retryLater(task)

		return
	}

	isCompleted := p.notifyAboutChanges(task, status, amount)
	if isCompleted {
		close(task.resultChan)
	} else {
		p.retryLater(task)
	}
}

func (p *poller) makeRequest(number string) (string, int64, error) {
	err := p.limiter.Wait(context.Background())
	if err != nil {
		return "", 0, fmt.Errorf("wait limiter failed: %w", err)
	}

	type accrualResponse struct {
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual"`
	}

	payload := new(accrualResponse)

	response, err := p.client.R().
		SetPathParam("number", number).
		SetResult(payload).
		Get("/api/orders/{number}")

	if err != nil {
		return "", 0, fmt.Errorf("cant make p request: %w", err)
	}

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

	return payload.Status, int64(payload.Accrual * 100), nil
}

func (p *poller) tuneRateLimiting(response *resty.Response) {
	p.tuningMutex.Lock()
	defer p.tuningMutex.Unlock()

	retryAfter, numberOfRequests, period := p.parseRateLimitedResponse(response)

	// block new requests until rate limitation has passed
	p.limiter.SetLimit(0)
	// tune limit according to response
	p.limiter.SetLimitAt(
		time.Now().Add(time.Duration(retryAfter)*time.Second),
		rate.Limit(numberOfRequests/period),
	)
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
	matches := regexp.MustCompile(`^No more than (\d+) requests per (second|minute|hour) allowed$`).FindAllString(body, 2)
	if len(matches) != 2 {
		p.logger.Errorw("cant parse rate limiting response, using default values", "body", body)

		return retryAfterSeconds, defaultNumberOfRequests, defaultPeriod
	}

	numberOfRequests, err := strconv.Atoi(matches[0])
	if err != nil {
		p.logger.Errorw("cant parse number of requests, using default value", "number", matches[0])

		numberOfRequests = defaultNumberOfRequests
	}

	period := defaultPeriod

	switch matches[1] {
	case "second":
		period = 1
	case "minute":
		period = 60
	case "hour":
		period = 60 * 60
	default:
		p.logger.Errorw("cant parse rate limiting period, using default values", "period", matches[1])
	}

	return numberOfRequests, period, numberOfRequests
}

func (p *poller) orderStatus(accrualStatus string) order.Status {
	switch accrualStatus {
	case "REGISTERED":
		return order.StatusNew
	case "INVALID":
		return order.StatusInvalid
	case "PROCESSING":
		return order.StatusProcessing
	case "PROCESSED":
		return order.StatusProcessed
	default:
		// when accrual returned 204 or something
		return order.StatusNew
	}
}

func (p *poller) retryLater(task *task) {
	after := p.calcRetryPeriod(task.attempts)

	time.AfterFunc(after, func() {
		err := p.enqueue(task)
		if err != nil {
			p.logger.Errorw("cant submit to task to pool", "number", task.number, "error", err)
		}
	})
}

func (p *poller) calcRetryPeriod(attempt int) time.Duration {
	// capped exponential backoff
	interval := math.Min(float64(p.maxRetryWaitTime), float64(time.Second)*math.Exp2(float64(attempt)))

	return time.Duration(interval)
}

func (p *poller) notifyAboutChanges(task *task, receivedStatus string, receivedAccrual int64) bool {
	orderStatus := p.orderStatus(receivedStatus)

	if orderStatus == order.StatusProcessed {
		task.resultChan <- order.AccrualResult{
			Status:  orderStatus,
			Accrual: &receivedAccrual,
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

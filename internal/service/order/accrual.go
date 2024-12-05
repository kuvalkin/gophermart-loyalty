package order

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

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/pool"
)

// todo refactor
type accrual struct {
	pool             *pool.Pool
	limiter          *rate.Limiter
	tuningMutex      *sync.Mutex
	client           *resty.Client
	maxAttempts      int
	maxRetryWaitTime time.Duration
	logger           *zap.SugaredLogger
}

type accrualResult struct {
	status  Status
	accrual *int64
	err     error
}

type accrualTask struct {
	number      string
	knownStatus Status
	attempts    int
	resultChan  chan accrualResult
}

func newAccrual(accrualSystemAddress string, maxRetries int, maxRetryWaitTime time.Duration) (*accrual, error) {
	// todo call Release on shutdown?
	p, err := pool.NewPool(nil)
	if err != nil {
		return nil, fmt.Errorf("cant create a new pool: %w", err)
	}

	return &accrual{
		pool:             p,
		limiter:          rate.NewLimiter(rate.Inf, 1),
		tuningMutex:      &sync.Mutex{},
		client:           resty.New().SetBaseURL(accrualSystemAddress),
		maxAttempts:      maxRetries,
		maxRetryWaitTime: maxRetryWaitTime,
		logger:           log.Logger().Named("accrual"),
	}, nil
}

func (a *accrual) addToQueue(number string, status Status) (<-chan accrualResult, error) {
	result := make(chan accrualResult, 1)

	task := &accrualTask{
		number:      number,
		knownStatus: status,
		resultChan:  result,
		attempts:    0,
	}

	err := a.enqueue(task)
	if err != nil {
		return nil, fmt.Errorf("cant submit to task to pool: %w", err)
	}

	return result, nil
}

func (a *accrual) enqueue(task *accrualTask) error {
	return a.pool.Submit(func() {
		a.processTask(task)
	})
}

func (a *accrual) processTask(task *accrualTask) {
	task.attempts++

	status, amount, err := a.makeRequest(task.number)
	if err != nil {
		if task.attempts > a.maxAttempts {
			task.resultChan <- accrualResult{
				err: fmt.Errorf("max attempts exceeded"),
			}

			close(task.resultChan)

			return
		}

		a.retryLater(task)

		return
	}

	isCompleted := a.notifyAboutChanges(task, status, amount)
	if isCompleted {
		close(task.resultChan)
	} else {
		a.retryLater(task)
	}
}

func (a *accrual) makeRequest(number string) (string, int64, error) {
	err := a.limiter.Wait(context.Background())
	if err != nil {
		return "", 0, fmt.Errorf("wait limiter failed: %w", err)
	}

	type accrualResponse struct {
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual"`
	}

	payload := new(accrualResponse)

	response, err := a.client.R().
		SetPathParam("number", number).
		SetResult(payload).
		Get("/api/orders/{number}")

	if err != nil {
		return "", 0, fmt.Errorf("cant make a request: %w", err)
	}

	if response.StatusCode() == http.StatusTooManyRequests {
		a.tuneRateLimiting(response)

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

func (a *accrual) tuneRateLimiting(response *resty.Response) {
	a.tuningMutex.Lock()
	defer a.tuningMutex.Unlock()

	retryAfter, numberOfRequests, period := a.parseRateLimitedResponse(response)

	// block new requests until rate limitation has passed
	a.limiter.SetLimit(0)
	// tune limit according to response
	a.limiter.SetLimitAt(
		time.Now().Add(time.Duration(retryAfter)*time.Second),
		rate.Limit(numberOfRequests/period),
	)
	// limit number of goroutines
	a.pool.Tune(numberOfRequests)
}

func (a *accrual) parseRateLimitedResponse(response *resty.Response) (int, int, int) {
	retryAfterHeader := response.Header().Get("Retry-After")
	retryAfterSeconds, err := strconv.Atoi(retryAfterHeader)
	if err != nil {
		a.logger.Warnw("cant parse Retry-After header, using default value", "header", retryAfterHeader)

		retryAfterSeconds = 60
	}

	defaultNumberOfRequests := 1
	defaultPeriod := 60

	body := strings.TrimSpace(response.String())
	matches := regexp.MustCompile(`^No more than (\d+) requests per (second|minute|hour) allowed$`).FindAllString(body, 2)
	if len(matches) != 2 {
		a.logger.Errorw("cant parse rate limiting response, using default values", "body", body)

		return retryAfterSeconds, defaultNumberOfRequests, defaultPeriod
	}

	numberOfRequests, err := strconv.Atoi(matches[0])
	if err != nil {
		a.logger.Errorw("cant parse number of requests, using default value", "number", matches[0])

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
		a.logger.Errorw("cant parse rate limiting period, using default values", "period", matches[1])
	}

	return numberOfRequests, period, numberOfRequests
}

func (a *accrual) orderStatus(accrualStatus string) Status {
	switch accrualStatus {
	case "REGISTERED":
		return StatusNew
	case "INVALID":
		return StatusInvalid
	case "PROCESSING":
		return StatusProcessing
	case "PROCESSED":
		return StatusProcessed
	default:
		// when accrual returned 204 or something
		return StatusNew
	}
}

func (a *accrual) retryLater(task *accrualTask) {
	after := a.calcRetryPeriod(task.attempts)

	time.AfterFunc(after, func() {
		err := a.enqueue(task)
		if err != nil {
			a.logger.Errorw("cant submit to task to pool", "number", task.number, "error", err)
		}
	})
}

func (a *accrual) calcRetryPeriod(attempt int) time.Duration {
	// capped exponential backoff
	interval := math.Min(float64(a.maxRetryWaitTime), float64(time.Second)*math.Exp2(float64(attempt)))

	return time.Duration(interval)
}

func (a *accrual) notifyAboutChanges(task *accrualTask, receivedStatus string, receivedAccrual int64) bool {
	orderStatus := a.orderStatus(receivedStatus)

	if orderStatus == StatusProcessed {
		task.resultChan <- accrualResult{
			status:  orderStatus,
			accrual: &receivedAccrual,
		}

		return true
	}

	if orderStatus != task.knownStatus {
		task.knownStatus = orderStatus
		task.resultChan <- accrualResult{
			status: orderStatus,
		}
	}

	return false
}

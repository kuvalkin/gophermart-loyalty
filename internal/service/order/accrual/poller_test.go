package accrual

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/test"
)

func TestPoller(t *testing.T) {
	log.InitTestLogger(t)

	t.Run("success", testPollerSuccess)
	t.Run("service says that order invalid", testPollerServiceSaysOrderInvalid)
	t.Run("service not responding", testPollerServiceNotResponding)
	t.Run("order is never registered", testPollerOrderIsNeverRegistered)
	t.Run("order status is never changed", testPollerOrderStatusIsNeverChanged)
	t.Run("order already enqueued", testPollerOrderAlreadyEnqueued)
	t.Run("rate limiting", testPollerRateLimiting)
}

func testPollerSuccess(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	endpointCallCount := new(atomic.Int32)
	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		endpointCallCount.Add(1)

		switch endpointCallCount.Load() {
		case 1:
			return ctx.JSON(accrualResponse{
				Status: string(statusRegistered),
			})
		case 2:
			return ctx.JSON(accrualResponse{
				Status: string(statusProcessing),
			})
		case 3:
			return ctx.JSON(accrualResponse{
				Status: string(statusProcessing),
			})
		default:
			return ctx.JSON(accrualResponse{
				Status:  string(statusProcessed),
				Accrual: 100.93,
			})
		}
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusNew)
	require.NoError(t, err)

	expectedResultsSequence := map[int]order.AccrualResult{
		// poller notifies only about *changes*
		1: {
			Status: order.StatusProcessing,
		},
		2: {
			Status:  order.StatusProcessed,
			Accrual: test.Int64Pointer(10093),
		},
	}

	resultCount := 0

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			resultCount++

			expected, ok := expectedResultsSequence[resultCount]
			require.True(t, ok)

			log.Logger().Debugw("test case", "count", resultCount, "expected", expected)

			assert.NoError(t, result.Err)
			assert.Equal(t, expected.Status, result.Status)

			if expected.Accrual == nil {
				assert.Nil(t, result.Accrual)
			} else {
				require.NotNil(t, result.Accrual)
				assert.Equal(t, *expected.Accrual, *result.Accrual)
			}
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func testPollerServiceSaysOrderInvalid(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	endpointCallCount := 0
	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		endpointCallCount++

		switch endpointCallCount {
		case 1:
			return ctx.JSON(accrualResponse{
				Status: string(statusRegistered),
			})
		case 2:
			return ctx.JSON(accrualResponse{
				Status: string(statusProcessing),
			})
		case 3:
			return ctx.JSON(accrualResponse{
				Status: string(statusProcessing),
			})
		default:
			return ctx.JSON(accrualResponse{
				Status: string(statusInvalid),
			})
		}
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusNew)
	require.NoError(t, err)

	expectedResultsSequence := map[int]order.AccrualResult{
		// poller notifies only about *changes*
		1: {
			Status: order.StatusProcessing,
		},
		2: {
			Status: order.StatusInvalid,
		},
	}

	resultCount := 0

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			resultCount++

			expected, ok := expectedResultsSequence[resultCount]
			require.True(t, ok)

			log.Logger().Debugw("test case", "count", resultCount, "expected", expected)

			assert.NoError(t, result.Err)
			assert.Equal(t, expected.Status, result.Status)

			if expected.Accrual == nil {
				assert.Nil(t, result.Accrual)
			} else {
				require.NotNil(t, result.Accrual)
				assert.Equal(t, *expected.Accrual, *result.Accrual)
			}
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func testPollerServiceNotResponding(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// never respond until request is aborted or test is done
		select {
		case <-ctx.Done():
			return
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusNew)
	require.NoError(t, err)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			require.Error(t, result.Err)
			require.Equal(t, "max attempts exceeded", result.Err.Error())
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func testPollerOrderIsNeverRegistered(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		// pretend that this order is unknown on every request
		return ctx.SendStatus(fiber.StatusNoContent)
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusNew)
	require.NoError(t, err)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			require.Error(t, result.Err)
			require.Equal(t, "max attempts exceeded", result.Err.Error())
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func testPollerOrderStatusIsNeverChanged(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		return ctx.JSON(accrualResponse{
			Status: string(statusProcessing),
		})
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusProcessing)
	require.NoError(t, err)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			require.Error(t, result.Err)
			require.Equal(t, "max attempts exceeded", result.Err.Error())
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func testPollerOrderAlreadyEnqueued(t *testing.T) {
	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		return ctx.JSON(accrualResponse{
			Status:  string(statusProcessed),
			Accrual: 0,
		})
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	number := test.NewOrderNumber()

	_, err := p.Enqueue(number, order.StatusProcessing)
	require.NoError(t, err)

	_, err = p.Enqueue(number, order.StatusProcessing)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already enqueued")
}

func testPollerRateLimiting(t *testing.T) {
	ctx, cancel := test.Context(t)
	defer cancel()

	isLimited := new(atomic.Bool)
	isLimited.Store(true)
	time.AfterFunc(time.Second, func() {
		isLimited.Store(false)
	})

	limitedRequestCount := new(atomic.Int32)
	defer func() {
		// First request to get a rate limitation answer.
		// Second request - because rate limiting is implemented as token bucket, and we adjust it refilling rate.
		// After the first request the bucket is almost certain to have a token
		if limitedRequestCount.Load() > 2 {
			log.Logger().Errorw("it seems that poller was not rate limited")
			t.FailNow()
		}
	}()

	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		if isLimited.Load() {
			limitedRequestCount.Add(1)

			log.Logger().Info("access to rate limited api")

			ctx.Set("Retry-After", "1")

			ctx.Status(fiber.StatusTooManyRequests)

			return ctx.SendString("No more than 1 requests per second allowed")
		}

		return ctx.JSON(accrualResponse{
			Status:  string(statusProcessed),
			Accrual: 0,
		})
	}))
	defer server.Close()

	p := newTestPoller(t, server)
	defer p.Close()

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusProcessing)
	require.NoError(t, err)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			require.NoError(t, result.Err)

		case <-ctx.Done():
			log.Logger().Errorw("ctx done", "error", ctx.Err())
			t.FailNow()
		}
	}
}

func newTestPoller(t *testing.T, server *httptest.Server) order.AccrualPoller {
	p, err := NewPoller(server.URL, 50*time.Millisecond, 5, time.Millisecond)
	require.NoError(t, err)

	return p
}

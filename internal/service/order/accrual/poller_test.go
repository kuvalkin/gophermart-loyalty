package accrual

import (
	"net/http/httptest"
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

	ctx, cancel := test.Context(t)
	defer cancel()

	server := httptest.NewServer(adaptor.FiberHandler(func(ctx *fiber.Ctx) error {
		return ctx.JSON(accrualResponse{
			Status:  string(statusProcessed),
			Accrual: 100.93,
		})
	}))
	defer server.Close()

	p, err := NewPoller(server.URL, 50*time.Millisecond, 1, time.Millisecond)
	require.NoError(t, err)

	resultChan, err := p.Enqueue(test.NewOrderNumber(), order.StatusNew)
	require.NoError(t, err)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Logger().Debug("result chan closed")

				return
			}

			assert.NoError(t, result.Err)
			assert.Equal(t, order.StatusProcessed, result.Status)
			assert.NotNil(t, result.Accrual)
			assert.Equal(t, int64(10093), *result.Accrual)
		case <-ctx.Done():
			log.Logger().Errorw("ctx done", err, ctx.Err())
			t.FailNow()
		}
	}
}

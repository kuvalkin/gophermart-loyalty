package balance

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/money"
	"github.com/kuvalkin/gophermart-loyalty/internal/test"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/handlerstest"
)

const balance = "/api/user/balance"
const withdraw = "/api/user/balance/withdraw"
const list = "/api/user/withdrawals"

func TestOrders(t *testing.T) {
	log.InitTestLogger(t)

	t.Run("auth", testAuth)

	t.Run("withdraw", func(t *testing.T) {
		t.Run("validation", testWithdrawValidation)
	})

	t.Run("flow", testBalanceFlow)
}

func testAuth(t *testing.T) {
	server := handlerstest.NewTestServer(t)
	defer server.Close()

	client := resty.New().SetBaseURL(server.URL)

	t.Run("request without token", func(t *testing.T) {
		t.Run("balance", func(t *testing.T) {
			response, err := client.R().Get(balance)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})

		t.Run("withdraw", func(t *testing.T) {
			response, err := client.R().
				SetBody(map[string]string{
					"order": test.NewOrderNumber(),
					"sum":   "12",
				}).
				Post(withdraw)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})

		t.Run("list", func(t *testing.T) {
			response, err := client.R().
				Get(list)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})
	})

	t.Run("request with invalid token", func(t *testing.T) {
		t.Run("balance", func(t *testing.T) {
			response, err := client.R().SetAuthToken("hi").Get(balance)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})

		t.Run("withdraw", func(t *testing.T) {
			response, err := client.R().
				SetAuthToken("hi").
				SetBody(
					map[string]string{
						"order": test.NewOrderNumber(),
						"sum":   "12",
					},
				).
				Post(withdraw)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})

		t.Run("list", func(t *testing.T) {
			response, err := client.R().
				SetAuthToken("hi").
				Get(list)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		})
	})
}

func testWithdrawValidation(t *testing.T) {
	server, token := handlerstest.NewTestServerWithLoggedInUser(t)
	defer server.Close()

	tests := []handlerstest.TCase{
		{
			Name:        "success",
			Token:       token,
			ContentType: "application/json",
			Body: map[string]any{
				"order": test.NewOrderNumber(),
				"sum":   12,
			},
			Want: handlerstest.Want{
				Status: http.StatusPaymentRequired,
			},
		},
		{
			Name:        "invalid number",
			Token:       token,
			ContentType: "application/json",
			Body: map[string]any{
				"order": "32",
				"sum":   12,
			},
			Want: handlerstest.Want{
				Status: http.StatusUnprocessableEntity,
			},
		},
		{
			Name:        "empty number",
			Token:       token,
			ContentType: "application/json",
			Body: map[string]any{
				"order": "",
				"sum":   12,
			},
			Want: handlerstest.Want{
				Status: http.StatusUnprocessableEntity,
			},
		},
		{
			Name:        "empty body",
			Token:       token,
			ContentType: "application/json",
			Body:        "",
			Want: handlerstest.Want{
				Status: http.StatusBadRequest,
			},
		},
		{
			Name:        "invalid sum",
			Token:       token,
			ContentType: "application/json",
			Body: map[string]any{
				"order": test.NewOrderNumber(),
				"sum":   -1,
			},
			Want: handlerstest.Want{
				Status: http.StatusBadRequest,
			},
		},
		{
			Name:        "zero sum",
			Token:       token,
			ContentType: "application/json",
			Body: map[string]any{
				"order": test.NewOrderNumber(),
				"sum":   0,
			},
			Want: handlerstest.Want{
				Status: http.StatusBadRequest,
			},
		},
	}

	handlerstest.TestEndpoint(t, server, tests, http.MethodPost, withdraw)
}

func testBalanceFlow(t *testing.T) {
	testServer, token := handlerstest.NewTestServerWithLoggedInUser(t)
	defer testServer.Close()

	client := resty.New().SetBaseURL(testServer.URL)

	type balanceResponse struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	t.Run("empty account", func(t *testing.T) {
		t.Run("get balance", func(t *testing.T) {
			p := new(balanceResponse)

			response, err := client.R().SetAuthToken(token).SetResult(p).Get(balance)

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode())
			assert.Equal(t, float64(0), p.Current)
			assert.Equal(t, float64(0), p.Withdrawn)
		})

		t.Run("withdraw", func(t *testing.T) {
			response, err := client.R().SetAuthToken(token).SetBody(map[string]any{
				"order": test.NewOrderNumber(),
				"sum":   100,
			}).Post(withdraw)

			require.NoError(t, err)
			assert.Equal(t, http.StatusPaymentRequired, response.StatusCode())
		})

		t.Run("list", func(t *testing.T) {
			response, err := client.R().SetAuthToken(token).Get(list)

			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, response.StatusCode())
		})
	})

	increment := handlerstest.IncreaseBalance(t, testServer, token)

	t.Run("account with balance", func(t *testing.T) {
		t.Run("get balance", func(t *testing.T) {
			p := new(balanceResponse)

			response, err := client.R().SetAuthToken(token).SetResult(p).Get(balance)

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode())
			assert.Equal(t, money.IntToFloat(increment), p.Current)
			assert.Equal(t, float64(0), p.Withdrawn)
		})

		successOrderNumber := test.NewOrderNumber()

		t.Run("withdraw", func(t *testing.T) {
			t.Run("sum too big", func(t *testing.T) {
				response, err := client.R().SetAuthToken(token).SetBody(map[string]any{
					"order": test.NewOrderNumber(),
					"sum":   money.IntToFloat(increment) * 1000,
				}).Post(withdraw)

				require.NoError(t, err)
				assert.Equal(t, http.StatusPaymentRequired, response.StatusCode())
			})

			t.Run("success", func(t *testing.T) {
				response, err := client.R().SetAuthToken(token).SetBody(map[string]any{
					"order": successOrderNumber,
					"sum":   money.IntToFloat(increment),
				}).Post(withdraw)

				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode())
			})
		})

		t.Run("list", func(t *testing.T) {
			response, err := client.R().SetAuthToken(token).Get(list)

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode())
			assert.JSONEq(
				t,
				fmt.Sprintf(
					`[{ "order": "%v", "sum": %v, "processed_at": "%v" }]`,
					successOrderNumber,
					money.IntToFloat(increment),
					time.Now().Format(time.RFC3339),
				),
				string(response.Body()),
			)
		})
	})

}

package orders

import (
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/test"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/handlerstest"
)

const orders = "/api/user/orders"

func TestOrders(t *testing.T) {
	log.InitTestLogger(t)

	t.Run("upload", func(t *testing.T) {
		t.Run("auth", testAuth)
		t.Run("validation", testUploadValidation)
		t.Run("flow", testUploadFlow)
	})
}

func testAuth(t *testing.T) {
	server := handlerstest.NewTestServer(t)
	defer server.Close()

	client := resty.New().SetBaseURL(server.URL)

	t.Run("request without token", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(test.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
	})

	t.Run("request with invalid token", func(t *testing.T) {
		response, err := client.R().
			SetAuthToken("hi").
			SetHeader("Content-Type", "text/plain").
			SetBody(test.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
	})
}

func testUploadValidation(t *testing.T) {
	server, token := handlerstest.NewTestServerWithLoggedInUser(t)
	defer server.Close()

	tests := []handlerstest.TCase{
		{
			Name:        "success",
			Token:       token,
			ContentType: "text/plain",
			Body:        test.NewOrderNumber(),
			Want: handlerstest.Want{
				Status: http.StatusAccepted,
			},
		},
		{
			Name:        "json",
			Token:       token,
			ContentType: "application/json",
			Body: handlerstest.JSON(t, map[string]string{
				"number": test.NewOrderNumber(),
			}),
			Want: handlerstest.Want{
				Status: http.StatusBadRequest,
			},
		},
		{
			Name:        "empty body",
			Token:       token,
			ContentType: "text/plain",
			Body:        "",
			Want: handlerstest.Want{
				Status: http.StatusUnprocessableEntity,
			},
		},
	}

	handlerstest.TestEndpoint(t, server, tests, http.MethodPost, orders)
}

func testUploadFlow(t *testing.T) {
	testServer, token1 := handlerstest.NewTestServerWithLoggedInUser(t)
	defer testServer.Close()

	token2 := handlerstest.LoginNewUser(t, testServer)

	client := resty.New().SetBaseURL(testServer.URL)

	number := test.NewOrderNumber()

	t.Run("upload new order", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetAuthToken(token1).
			SetBody(number).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, response.StatusCode())
	})

	t.Run("upload order with this number again by same user", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetAuthToken(token1).
			SetBody(number).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode())
	})

	t.Run("another user uploads order with same number", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetAuthToken(token2).
			SetBody(number).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, response.StatusCode())
	})

	t.Run("another user uploads order with another number", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetAuthToken(token2).
			SetBody(test.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, response.StatusCode())
	})
}

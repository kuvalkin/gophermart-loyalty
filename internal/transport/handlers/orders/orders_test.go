package orders

import (
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	test2 "github.com/kuvalkin/gophermart-loyalty/internal/test"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/test"
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
	server := test.NewTestServer(t)
	defer server.Close()

	client := resty.New().SetBaseURL(server.URL)

	t.Run("request without token", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetBody(test2.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
	})

	t.Run("request with invalid token", func(t *testing.T) {
		response, err := client.R().
			SetAuthToken("hi").
			SetHeader("Content-Type", "text/plain").
			SetBody(test2.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
	})
}

func testUploadValidation(t *testing.T) {
	server, token := test.NewTestServerWithLoggedInUser(t)
	defer server.Close()

	tests := []test.TCase{
		{
			Name:        "success",
			Token:       token,
			ContentType: "text/plain",
			Body:        test2.NewOrderNumber(),
			Want: test.Want{
				Status: http.StatusAccepted,
			},
		},
		{
			Name:        "json",
			Token:       token,
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"number": test2.NewOrderNumber(),
			}),
			Want: test.Want{
				Status: http.StatusBadRequest,
			},
		},
		{
			Name:        "empty body",
			Token:       token,
			ContentType: "text/plain",
			Body:        "",
			Want: test.Want{
				Status: http.StatusUnprocessableEntity,
			},
		},
	}

	test.TestEndpoint(t, server, tests, http.MethodPost, orders)
}

func testUploadFlow(t *testing.T) {
	testServer, token1 := test.NewTestServerWithLoggedInUser(t)
	defer testServer.Close()

	token2 := test.LoginNewUser(t, testServer)

	client := resty.New().SetBaseURL(testServer.URL)

	number := test2.NewOrderNumber()

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
			SetBody(test2.NewOrderNumber()).
			Post(orders)

		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, response.StatusCode())
	})
}

package auth

import (
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/test"
)

const register = "/api/user/register"
const login = "/api/user/login"

func TestAuth(t *testing.T) {
	log.InitTestLogger(t)

	t.Run("register", func(t *testing.T) {
		t.Run("payload validation", testRegisterPayloadValidation)
	})

	t.Run("login", func(t *testing.T) {
		t.Run("payload validation", testLoginPayloadValidation)
	})

	t.Run("flow", testFlow)
}

func testRegisterPayloadValidation(t *testing.T) {
	tests := []test.TCase{
		{
			Name:        "positive",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "test",
				"password": "longmegapassword",
			}),
			Want: test.Want{
				Status:      200,
				ContentType: "application/json",
			},
		},
		{
			Name:        "empty string",
			ContentType: "application/json",
			Want: test.Want{
				Status:      400,
				Body:        "Bad Request",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "invalid string",
			ContentType: "application/json",
			Body:        "hi",
			Want: test.Want{
				Status:      400,
				Body:        "Bad Request",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "empty login",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "",
				"password": "longmegapassword",
			}),
			Want: test.Want{
				Status:      400,
				Body:        "invalid login",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "empty password",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "test",
				"password": "",
			}),
			Want: test.Want{
				Status:      400,
				Body:        "password is too short",
				ContentType: "text/plain; charset=utf-8",
			},
		},
	}

	server := test.NewTestServer(t)
	defer server.Close()

	test.TestEndpoint(t, server, tests, http.MethodPost, register)
}

func testFlow(t *testing.T) {
	testServer := test.NewTestServer(t)
	defer testServer.Close()
	client := resty.New().SetBaseURL(testServer.URL)

	payload := test.JSON(t, map[string]string{
		"login":    "hi",
		"password": "longmegapassword",
	})

	type result struct {
		Token string `json:"token"`
	}

	t.Run("register new user", func(t *testing.T) {
		r := new(result)

		response, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(payload).
			SetResult(r).
			Post(register)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode())
		assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
		assert.NotEmpty(t, r.Token)
	})

	t.Run("register but login taken", func(t *testing.T) {
		response, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(payload).
			Post(register)

		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, response.StatusCode())
	})

	t.Run("login", func(t *testing.T) {
		r := new(result)

		response, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(payload).
			SetResult(r).
			Post(login)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode())
		assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
		assert.NotEmpty(t, r.Token)
	})
}

func testLoginPayloadValidation(t *testing.T) {
	tests := []test.TCase{
		{
			Name:        "positive",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "test",
				"password": "longmegapassword",
			}),
			Want: test.Want{
				Status:      401, // since this user not exists
				Body:        "Unauthorized",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "empty string",
			ContentType: "application/json",
			Want: test.Want{
				Status:      400,
				Body:        "Bad Request",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "invalid string",
			ContentType: "application/json",
			Body:        "hi",
			Want: test.Want{
				Status:      400,
				Body:        "Bad Request",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "empty login",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "",
				"password": "longmegapassword",
			}),
			Want: test.Want{
				Status:      401,
				Body:        "Unauthorized",
				ContentType: "text/plain; charset=utf-8",
			},
		},
		{
			Name:        "empty password",
			ContentType: "application/json",
			Body: test.JSON(t, map[string]string{
				"login":    "test",
				"password": "",
			}),
			Want: test.Want{
				Status:      401,
				Body:        "Unauthorized",
				ContentType: "text/plain; charset=utf-8",
			},
		},
	}

	server := test.NewTestServer(t)
	defer server.Close()

	test.TestEndpoint(t, server, tests, http.MethodPost, login)
}

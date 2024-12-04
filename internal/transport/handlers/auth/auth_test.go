package auth

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/internal/test"
)

const register = "/api/user/register"
const login = "/api/user/login"

type want struct {
	status      int
	body        string
	contentType string
}

type tcase struct {
	name string
	body string
	want
}

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
	tests := []tcase{
		{
			"positive",
			test.JSON(t, map[string]string{
				"login":    "test",
				"password": "longmegapassword",
			}),
			want{
				200,
				"",
				"application/json",
			},
		},
		{
			"empty string",
			"",
			want{
				400,
				"Bad Request",
				"text/plain; charset=utf-8",
			},
		},
		{
			"invalid string",
			"hi",
			want{
				400,
				"Bad Request",
				"text/plain; charset=utf-8",
			},
		},
		{
			"empty login",
			test.JSON(t, map[string]string{
				"login":    "",
				"password": "longmegapassword",
			}),
			want{
				400,
				"invalid login",
				"text/plain; charset=utf-8",
			},
		},
		{
			"empty password",
			test.JSON(t, map[string]string{
				"login":    "test",
				"password": "",
			}),
			want{
				400,
				"password is too short",
				"text/plain; charset=utf-8",
			},
		},
	}

	testEndpoint(t, tests, register)
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
	tests := []tcase{
		{
			"positive",
			test.JSON(t, map[string]string{
				"login":    "test",
				"password": "longmegapassword",
			}),
			want{
				401, // since this user not exists
				"Unauthorized",
				"text/plain; charset=utf-8",
			},
		},
		{
			"empty string",
			"",
			want{
				400,
				"Bad Request",
				"text/plain; charset=utf-8",
			},
		},
		{
			"invalid string",
			"hi",
			want{
				400,
				"Bad Request",
				"text/plain; charset=utf-8",
			},
		},
		{
			"empty login",
			test.JSON(t, map[string]string{
				"login":    "",
				"password": "longmegapassword",
			}),
			want{
				401,
				"Unauthorized",
				"text/plain; charset=utf-8",
			},
		},
		{
			"empty password",
			test.JSON(t, map[string]string{
				"login":    "test",
				"password": "",
			}),
			want{
				401,
				"Unauthorized",
				"text/plain; charset=utf-8",
			},
		},
	}

	testEndpoint(t, tests, login)
}

func testEndpoint(t *testing.T, cases []tcase, endpoint string) {
	testServer := test.NewTestServer(t)
	defer testServer.Close()
	client := resty.New().SetBaseURL(testServer.URL)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.R().
				SetBody(tt.body).
				SetHeader("Content-Type", "application/json").
				Post(endpoint)

			require.NoError(t, err)

			assert.Equal(t, tt.want.status, response.StatusCode())
			assert.Equal(t, tt.want.contentType, response.Header().Get("Content-Type"))
			if tt.want.body != "" {
				if strings.Contains(tt.want.contentType, "json") {
					assert.JSONEq(t, tt.want.body, string(response.Body()))
				} else {
					assert.Contains(t, string(response.Body()), tt.want.body)
				}
			}
		})
	}
}

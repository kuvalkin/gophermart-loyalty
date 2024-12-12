package handlerstest

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	balanceStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/storage/balance/withdrawals"
	orderStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/order"
	userStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport"
)

func NewTestServer(t *testing.T) *httptest.Server {
	server := transport.NewServer(defaultTestConfig(), &transport.Services{
		User:    userService(t),
		Order:   orderService(),
		Balance: balanceService(t),
	})

	return server.NewTestServer()
}

func LoginNewUser(t *testing.T, server *httptest.Server) string {
	login := rand.Int()
	// ensure adequate password length
	password := 12345678910 + rand.Int()

	type payload struct {
		Token string `json:"token"`
	}
	result := new(payload)

	resp, err := resty.New().SetBaseURL(server.URL).R().
		SetBody(map[string]string{
			"login":    strconv.Itoa(login),
			"password": strconv.Itoa(password),
		}).
		SetResult(result).
		Post("/api/user/register")

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotEmpty(t, result.Token)

	return result.Token
}

func NewTestServerWithLoggedInUser(t *testing.T) (*httptest.Server, string) {
	server := NewTestServer(t)

	token := LoginNewUser(t, server)

	return server, token
}

func userService(t *testing.T) user.Service {
	conf := defaultTestConfig()

	service, err := user.NewService(userStorage.NewInMemoryRepository(), &user.Options{
		TokenSecret:           []byte("test"),
		PasswordSalt:          "test",
		MinPasswordLength:     conf.MinPasswordLength,
		TokenExpirationPeriod: conf.TokenExpirationPeriod,
	})
	require.NoError(t, err)

	return service
}

func orderService() order.Service {
	return order.NewService(orderStorage.NewInMemoryRepository(), newDummyPoller())
}

func balanceService(t *testing.T) balance.Service {
	b, err := balance.NewService(balanceStorage.NewInMemoryRepository(), withdrawals.NewMemoryRepository(), newDummyTxProvider())
	require.NoError(t, err)

	return b
}

func defaultTestConfig() *config.Config {
	return &config.Config{
		RunAddress:            "",
		DatabaseDSN:           "",
		DatabaseTimeout:       time.Second,
		AccrualSystemAddress:  "",
		MinPasswordLength:     12,
		TokenExpirationPeriod: time.Hour,
	}
}

func JSON(t *testing.T, payload map[string]string) string {
	buf, err := json.Marshal(payload)
	require.NoError(t, err)

	return string(buf)
}

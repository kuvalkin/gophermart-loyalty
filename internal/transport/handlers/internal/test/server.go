package test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	orderStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/order"
	userStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport"
)

func NewTestServer(t *testing.T) *httptest.Server {
	server := transport.NewServer(defaultTestConfig(), &transport.Services{
		User:  userService(t),
		Order: orderService(),
	})

	return server.NewTestServer()
}

const testUserLogin = "my_test_user"
const testUserPassword = "my_mega_test_password"

func NewTestServerWithLoggedInUser(t *testing.T) (*httptest.Server, string) {
	us := userService(t)

	err := us.Register(context.Background(), testUserLogin, testUserPassword)
	require.NoError(t, err)

	token, err := us.Login(context.Background(), testUserLogin, testUserPassword)
	require.NoError(t, err)

	server := transport.NewServer(defaultTestConfig(), &transport.Services{
		User:  userService(t),
		Order: orderService(),
	})

	return server.NewTestServer(), token
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
	// todo unit test on poller/order integration
	return order.NewService(orderStorage.NewInMemoryRepository(), newDummyPoller())
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

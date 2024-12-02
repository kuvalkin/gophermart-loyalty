package test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	userStorage "github.com/kuvalkin/gophermart-loyalty/internal/storage/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport"
)

func NewTestServer(t *testing.T) *httptest.Server {
	conf := defaultTestConfig()

	userService, err := user.NewService(userStorage.NewInMemoryRepository(), &user.Options{
		TokenSecret:           []byte("test"),
		PasswordSalt:          "test",
		MinPasswordLength:     conf.MinPasswordLength,
		TokenExpirationPeriod: conf.TokenExpirationPeriod,
	})

	require.NoError(t, err)

	server := transport.NewServer(conf, &transport.Services{
		User: userService,
	})

	return server.NewTestServer()
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

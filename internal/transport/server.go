package transport

import (
	"context"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

type Server struct {
	app     *fiber.App
	address string
}

type Services struct {
	User    user.Service
	Order   order.Service
	Balance balance.Service
}

func NewServer(conf *config.Config, services *Services) *Server {
	app := createAppWithRoutes(services)

	return &Server{app: app, address: conf.RunAddress}
}

// NewTestServer obviously should not be used in production code
func (s *Server) NewTestServer() *httptest.Server {
	return httptest.NewServer(adaptor.FiberApp(s.app))
}

func (s *Server) ListenAndServe() error {
	log.Logger().Infow("starting server", "address", s.address)

	return s.app.Listen(s.address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Logger().Info("shutting down server")

	return s.app.ShutdownWithContext(ctx)
}

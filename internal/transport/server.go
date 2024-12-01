package transport

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
)

type Server struct {
	app     *fiber.App
	address string
}

type Services struct {
	User user.Service
}

func NewServer(conf *config.Config, services *Services) *Server {
	app := createAppWithRoutes(services)

	return &Server{app: app, address: conf.RunAddress}
}

func (s *Server) ListenAndServe() error {
	log.Logger().Infow("starting server", "address", s.address)

	return s.app.Listen(s.address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Logger().Info("shutting down server")

	return s.app.ShutdownWithContext(ctx)
}

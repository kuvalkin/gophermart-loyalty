package transport

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/log"
)

type Server struct {
	app     *fiber.App
	address string
}

func NewServer(conf *config.Config) *Server {
	app := createAppWithRoutes(conf)

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

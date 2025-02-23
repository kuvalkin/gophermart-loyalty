package transport

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/auth/login"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/auth/register"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/balance/get"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/balance/withdraw"
	withdrawalsList "github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/balance/withdraw/list"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/orders/list"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/orders/upload"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/middleware/auth"
)

func createAppWithRoutes(services *Services) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:            "gophermart-loyalty",
		EnableIPValidation: true,
		Immutable:          true,
	})

	globalMiddleware(app)
	routes(app, services)

	return app
}

func globalMiddleware(app *fiber.App) {
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} - ${locals:requestid} | ${method} | ${path} | ${error}\n",
	}))
	app.Use(recover.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())
}

func routes(app *fiber.App, services *Services) {
	apiGroup := app.Group("/api")

	userGroup := apiGroup.Group("/user")

	userGroup.Post("/register", register.New(services.User).Handle)
	userGroup.Post("/login", login.New(services.User).Handle)

	authMiddleware := auth.New(services.User)

	userGroup.Post("/orders", authMiddleware, upload.New(services.Order).Handle)
	userGroup.Get("/orders", authMiddleware, list.New(services.Order).Handle)

	userGroup.Get("/balance", authMiddleware, get.New(services.Balance).Handle)
	userGroup.Post("/balance/withdraw", authMiddleware, withdraw.New(services.Balance).Handle)
	userGroup.Get("/withdrawals", authMiddleware, withdrawalsList.New(services.Balance).Handle)
}

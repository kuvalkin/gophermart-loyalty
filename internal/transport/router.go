package transport

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/auth/login"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/auth/register"
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

	//todo real handlers
	stub := func(ctx *fiber.Ctx) error {
		return ctx.SendString(fmt.Sprintf("%v %v", ctx.Method(), ctx.Path()))
	}

	userGroup.Post("/register", register.New(services.User).Handle)
	userGroup.Post("/login", login.New(services.User).Handle)

	authMiddleware := auth.New(services.User)

	userGroup.Post("/orders", authMiddleware, upload.New(services.Orders).Handle)
	userGroup.Get("/orders", authMiddleware, stub)

	userGroup.Get("/balance", authMiddleware, stub)
	userGroup.Post("/balance/withdraw", authMiddleware, stub)
	userGroup.Get("/withdrawals", authMiddleware, stub)
}

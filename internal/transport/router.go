package transport

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/kuvalkin/gophermart-loyalty/internal/config"
	"github.com/kuvalkin/gophermart-loyalty/internal/log"
)

func createAppWithRoutes(conf *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:            "gophermart-loyalty",
		EnableIPValidation: true,
	})

	globalMiddleware(app)
	routes(app, conf)

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

func routes(app *fiber.App, conf *config.Config) {
	api := app.Group("/api")

	user := api.Group("/user")

	stub := func(ctx *fiber.Ctx) error {
		return ctx.SendString(fmt.Sprintf("%v %v", ctx.Method(), ctx.Path()))
	}

	//todo real handlers
	user.Post("/register", stub)
	user.Post("/login", stub)

	auth := func(ctx *fiber.Ctx) error {
		log.Logger().Info("auth check")
		return ctx.Next()
	}

	user.Post("/orders", auth, stub)
	user.Get("/orders", auth, stub)

	user.Get("/balance", auth, stub)
	user.Post("/balance/withdraw", auth, stub)
	user.Get("/withdrawals", auth, stub)
}

package transport

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/kuvalkin/gophermart-loyalty/internal/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
)

const requestIdKey = "requestid"

func createAppWithRoutes(services *Services) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:            "gophermart-loyalty",
		EnableIPValidation: true,
	})

	globalMiddleware(app)
	routes(app, services)

	return app
}

func globalMiddleware(app *fiber.App) {
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} - ${locals:" + requestIdKey + "} | ${method} | ${path} | ${error}\n",
	}))
	app.Use(recover.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())
}

func routes(app *fiber.App, services *Services) {
	apiGroup := app.Group("/api")

	userGroup := apiGroup.Group("/user")

	stub := func(ctx *fiber.Ctx) error {
		return ctx.SendString(fmt.Sprintf("%v %v", ctx.Method(), ctx.Path()))
	}

	//todo real handlers
	userGroup.Post("/register", stub)
	userGroup.Post("/login", stub)

	auth := newAuth(services.User)

	userGroup.Post("/orders", auth, stub)
	userGroup.Get("/orders", auth, stub)

	userGroup.Get("/balance", auth, stub)
	userGroup.Post("/balance/withdraw", auth, stub)
	userGroup.Get("/withdrawals", auth, stub)
}

func newAuth(userService user.Service) func(ctx *fiber.Ctx) error {
	authLogger := log.Logger().Named("auth")

	return func(ctx *fiber.Ctx) error {
		authRequestLogger := authLogger.WithLazy("requestId", ctx.Locals(requestIdKey))

		authSlice, ok := ctx.GetReqHeaders()["Authorization"]
		if !ok {
			authRequestLogger.Debug("no Authorization header")

			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		if len(authSlice) != 1 {
			authRequestLogger.Debugw("invalid Authorization header", "header", authSlice)

			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		bearer := authSlice[0]

		token, found := strings.CutPrefix(bearer, "Bearer ")
		if !found {
			authRequestLogger.Debugw("Authorization header is not Bearer format", "header", bearer)

			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		err := userService.CheckToken(ctx.Context(), token)
		if err != nil {
			authRequestLogger.Debugw("token check failed", "error", err)

			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		return ctx.Next()
	}
}

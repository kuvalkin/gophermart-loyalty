package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

func New(userService user.Service) func(ctx *fiber.Ctx) error {
	authLogger := log.Logger().Named("auth")

	return func(ctx *fiber.Ctx) error {
		authRequestLogger := authLogger.WithLazy("requestId", ctx.Locals("requestid"))

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

		//todo set locals userid from token
		err := userService.CheckToken(ctx.Context(), token)
		if err != nil {
			authRequestLogger.Debugw("token check failed", "error", err)

			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		return ctx.Next()
	}
}

package internal

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
)

func Login(ctx *fiber.Ctx, userService user.Service, login string, password string) error {
	token, err := userService.Login(ctx.Context(), login, password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidPair) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		if errors.Is(err, user.ErrInternal) {
			return ctx.SendStatus(fiber.StatusInternalServerError)
		}

		ctx.Status(fiber.StatusBadRequest)

		return ctx.SendString(err.Error())
	}

	ctx.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(fiber.Map{"token": token})
}

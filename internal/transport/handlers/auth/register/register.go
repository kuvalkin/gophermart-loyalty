package register

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/transport/handlers/auth/internal"
)

type Handler struct {
	userService user.Service
}

func New(userService user.Service) *Handler {
	return &Handler{
		userService: userService,
	}
}

type payload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	p := new(payload)

	if err := ctx.BodyParser(p); err != nil {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	err := h.userService.Register(ctx.Context(), p.Login, p.Password)
	if err != nil {
		if errors.Is(err, user.ErrLoginTaken) {
			ctx.Status(fiber.StatusConflict)

			return ctx.SendString(err.Error())
		}

		if errors.Is(err, user.ErrInternal) {
			return ctx.SendStatus(fiber.StatusInternalServerError)
		}

		ctx.Status(fiber.StatusBadRequest)

		return ctx.SendString(err.Error())
	}

	return internal.Login(ctx, h.userService, p.Login, p.Password)
}

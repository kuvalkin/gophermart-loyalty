package login

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/user"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
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
		log.Logger().Debugw("invalid payload", "err", err, "requestid", ctx.Locals("requestid"))

		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	return internal.Login(ctx, h.userService, p.Login, p.Password)
}

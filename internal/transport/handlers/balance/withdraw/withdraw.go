package withdraw

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/balance"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/money"
)

type Handler struct {
	service balance.Service
}

func New(service balance.Service) *Handler {
	return &Handler{
		service: service,
	}
}

type payload struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	userIDRaw := ctx.Locals("userid")
	userID, ok := userIDRaw.(string)
	if !ok {
		log.Logger().Fatalw("no user id", "userIDRaw", userIDRaw)
		panic("no user id")
	}

	p := new(payload)

	if err := ctx.BodyParser(p); err != nil {
		log.Logger().Debugw("invalid payload", "err", err, "requestid", ctx.Locals("requestid"))

		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	err := h.service.Withdraw(ctx.Context(), userID, p.OrderNumber, money.FloatToInt(p.Sum))

	if errors.Is(err, balance.ErrNotEnoughBalance) {
		return ctx.SendStatus(fiber.StatusPaymentRequired)
	} else if errors.Is(err, balance.ErrInvalidOrderNumber) {
		return ctx.SendStatus(fiber.StatusUnprocessableEntity)
	} else if errors.Is(err, balance.ErrInvalidWithdrawalSum) {
		ctx.Status(fiber.StatusBadRequest)

		return ctx.SendString(err.Error())
	} else if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.SendStatus(fiber.StatusOK)
}

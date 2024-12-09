package get

import (
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

type balanceJSON struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	userIDRaw := ctx.Locals("userid")
	userID, ok := userIDRaw.(string)
	if !ok {
		log.Logger().Fatalw("no user id", "userIDRaw", userIDRaw)
		panic("no user id")
	}

	b, err := h.service.Get(ctx.Context(), userID)

	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(balanceJSON{
		Current:   money.IntToFloat(b.Current),
		Withdrawn: money.IntToFloat(b.Withdrawn),
	})
}

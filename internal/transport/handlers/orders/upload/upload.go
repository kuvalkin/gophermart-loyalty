package upload

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

type Handler struct {
	orderService order.Service
}

func New(orderService order.Service) *Handler {
	return &Handler{
		orderService: orderService,
	}
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	userIDRaw := ctx.Locals("userid")
	userID, ok := userIDRaw.(string)
	if !ok {
		log.Logger().Fatalw("no user id", "userIDRaw", userIDRaw)
		panic("no user id")
	}

	if !strings.HasPrefix(ctx.Get("Content-Type"), "text/plain") {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	body := strings.TrimSpace(string(ctx.Body()))

	err := h.orderService.Upload(ctx.Context(), userID, body)
	if errors.Is(err, order.ErrAlreadyUploaded) {
		return ctx.SendStatus(fiber.StatusOK)
	} else if errors.Is(err, order.ErrUploadedByAnotherUser) {
		return ctx.SendStatus(fiber.StatusConflict)
	} else if errors.Is(err, order.ErrInvalidNumber) {
		return ctx.SendStatus(fiber.StatusUnprocessableEntity)
	} else if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

package list

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
	"github.com/kuvalkin/gophermart-loyalty/internal/support/money"
)

type Handler struct {
	orderService order.Service
}

func New(orderService order.Service) *Handler {
	return &Handler{
		orderService: orderService,
	}
}

type orderJSON struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	userIDRaw := ctx.Locals("userid")
	userID, ok := userIDRaw.(string)
	if !ok {
		log.Logger().Fatalw("no user id", "userIDRaw", userIDRaw)
		panic("no user id")
	}

	list, err := h.orderService.List(ctx.Context(), userID)

	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	if len(list) == 0 {
		ctx.Status(fiber.StatusNoContent)
	} else {
		ctx.Status(fiber.StatusOK)
	}

	json := mapOrdersToJson(list)

	return ctx.JSON(json)
}

func mapOrdersToJson(orders []*order.Order) []*orderJSON {
	result := make([]*orderJSON, 0, len(orders))

	for _, o := range orders {
		singleResult := &orderJSON{
			Number:     o.Number,
			Status:     string(o.Status),
			UploadedAt: o.UploadedAt.Format(time.RFC3339),
		}

		if o.Accrual != nil {
			floatAccrual := money.IntToFloat(*o.Accrual)

			singleResult.Accrual = &floatAccrual
		}

		result = append(result, singleResult)
	}

	return result
}

package list

import (
	"time"

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

type historyEntryJson struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func (h *Handler) Handle(ctx *fiber.Ctx) error {
	userIDRaw := ctx.Locals("userid")
	userID, ok := userIDRaw.(string)
	if !ok {
		log.Logger().Fatalw("no user id", "userIDRaw", userIDRaw)
		panic("no user id")
	}

	list, err := h.service.WithdrawalHistory(ctx.Context(), userID)

	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	if len(list) == 0 {
		ctx.Status(fiber.StatusNoContent)
	} else {
		ctx.Status(fiber.StatusOK)
	}

	json := mapEntriesToJson(list)

	return ctx.JSON(json)
}

func mapEntriesToJson(entries []*balance.WithdrawalHistoryEntry) []*historyEntryJson {
	result := make([]*historyEntryJson, 0, len(entries))

	for _, e := range entries {
		singleResult := &historyEntryJson{
			OrderNumber: e.OrderNumber,
			Sum:         money.IntToFloat(e.Sum),
			ProcessedAt: e.ProcessedAt.Format(time.RFC3339),
		}

		result = append(result, singleResult)
	}

	return result
}

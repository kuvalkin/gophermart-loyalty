package accrual

import (
	"fmt"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
)

type accrualResponse struct {
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func statusFromString(status string) (accrualStatus, error) {
	switch status {
	case string(statusRegistered):
		return statusRegistered, nil
	case string(statusInvalid):
		return statusInvalid, nil
	case string(statusProcessing):
		return statusProcessing, nil
	case string(statusProcessed):
		return statusProcessed, nil
	default:
		return statusInvalid, fmt.Errorf("unknown accrualStatus: %s", status)
	}
}

type accrualStatus string

const statusRegistered = accrualStatus("REGISTERED")
const statusInvalid = accrualStatus("INVALID")
const statusProcessing = accrualStatus("PROCESSING")
const statusProcessed = accrualStatus("PROCESSED")

func (s accrualStatus) orderStatus() order.Status {
	switch s {
	case statusRegistered:
		return order.StatusNew
	case statusInvalid:
		return order.StatusInvalid
	case statusProcessing:
		return order.StatusProcessing
	case statusProcessed:
		return order.StatusProcessed
	default:
		// when accrual returned 204 or something
		return order.StatusNew
	}
}

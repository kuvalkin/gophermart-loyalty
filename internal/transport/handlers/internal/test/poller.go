package test

import "github.com/kuvalkin/gophermart-loyalty/internal/service/order"

func newDummyPoller() order.AccrualPoller {
	return &dummyPoller{}
}

type dummyPoller struct{}

func (p *dummyPoller) Enqueue(_ string, _ order.Status) (<-chan order.AccrualResult, error) {
	result := make(chan order.AccrualResult, 1)

	defer func() {
		var accrual int64 = 100

		result <- order.AccrualResult{
			Status:  order.StatusProcessed,
			Accrual: &accrual,
		}

		close(result)
	}()

	return result, nil
}

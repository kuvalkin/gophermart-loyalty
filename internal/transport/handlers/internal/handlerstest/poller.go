package handlerstest

import "github.com/kuvalkin/gophermart-loyalty/internal/service/order"

func newDummyPoller() order.AccrualPoller {
	return &dummyPoller{}
}

type dummyPoller struct{}

const ProcessedOrderAccrual int64 = 10093
const ProcessedOrderAccrualFloat float64 = 100.93

func (p *dummyPoller) Enqueue(_ string, _ order.Status) (<-chan order.AccrualResult, error) {
	result := make(chan order.AccrualResult, 1)

	defer func() {
		accrual := ProcessedOrderAccrual

		result <- order.AccrualResult{
			Status:  order.StatusProcessed,
			Accrual: &accrual,
		}

		close(result)
	}()

	return result, nil
}

func (p *dummyPoller) Close() error {
	return nil
}

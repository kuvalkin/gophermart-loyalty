package order

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kuvalkin/gophermart-loyalty/internal/service/order"
)

type UnprocessedOrder struct {
	Number        string
	CurrentStatus order.Status
}

func GetUnprocessedOrders(ctx context.Context, db *sql.DB) ([]*UnprocessedOrder, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT number, status FROM orders WHERE status IN ($1, $2)`,
		string(order.StatusNew),
		string(order.StatusProcessed),
	)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	defer rows.Close()

	result := make([]*UnprocessedOrder, 0)
	for rows.Next() {
		var number string
		var status string

		if err := rows.Scan(&number, &status); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		result = append(result, &UnprocessedOrder{
			Number:        number,
			CurrentStatus: order.Status(status),
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

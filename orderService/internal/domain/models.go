package domain

import "time"

type CreateOrderRequest struct {
	CustomerName    string            `json:"customer_name"`
	Type            string            `json:"order_type"`
	TableNumber     int               `json:"table_number"`
	DeliveryAddress string            `json:"delivery_address"`
	Items           []CreateOrderItem `json:"items"`
}

type CreateOrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type OrderResponce struct {
	Number      string  `db:"number"`
	Status      string  `db:"status"`
	TotalAmount float64 `db:"total_amount"`
}

type OrderStatusLog struct {
	ID        int        `db:"id"`
	CreatedAt time.Time  `db:"created_at"`
	OrderID   int        `db:"order_id"`
	Status    string     `db:"status"`
	ChangedBy string     `db:"changed_by"`
	ChangedAt *time.Time `db:"changed_at"`
	Notes     *string    `db:"notes"`
}

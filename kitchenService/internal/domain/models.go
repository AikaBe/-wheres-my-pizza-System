// internal/domain/models.go
package domain

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderMessage struct {
	OrderNumber     string      `json:"order_number"`
	CustomerName    string      `json:"customer_name"`
	OrderType       string      `json:"order_type"`
	TableNumber     int         `json:"table_number,omitempty"`
	DeliveryAddress string      `json:"delivery_address,omitempty"`
	Items           []OrderItem `json:"items"`
	TotalAmount     float64     `json:"total_amount"`
	Priority        int         `json:"priority"`
}

type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type StatusUpdateMessage struct {
	OrderNumber         string    `json:"order_number"`
	OldStatus           string    `json:"old_status"`
	NewStatus           string    `json:"new_status"`
	ChangedBy           string    `json:"changed_by"`
	Timestamp           time.Time `json:"timestamp"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
}

type KitchenRepository interface {
	RegisterWorker(ctx context.Context, workerName, workerType string) error
	UpdateWorkerHeartbeat(ctx context.Context, workerName string) error
	SetWorkerOffline(ctx context.Context, workerName string) error
	UpdateOrderStatus(ctx context.Context, orderNumber, status, processedBy string) error
	CompleteOrder(ctx context.Context, orderNumber, processedBy string) error
	GetOrderCurrentStatus(ctx context.Context, orderNumber string) (string, error)
}

type RabbitMQConsumer interface {
	ConsumeMessages(ctx context.Context, queueName string, prefetch int, handler func(msg amqp.Delivery) error) error
	PublishStatusUpdate(ctx context.Context, update StatusUpdateMessage) error
	Close() error
}

package domain

import "time"

// Сообщение, которое приходит из RabbitMQ
type OrderMessage struct {
	OrderID   string `json:"order_number"`
	OrderType string `json:"order_type"`
	Status    string `json:"status"`
	Worker    string `json:"worker"`
}

// Воркер (повар), регистрируется в БД
type Worker struct {
	ID              int
	CreatedAt       time.Time
	Name            string
	Type            string
	Status          string
	LastSeen        time.Time
	OrdersProcessed int
}

// Уведомление о смене статуса
type StatusNotification struct {
	OrderNumber         string `json:"order_number"`
	OldStatus           string `json:"old_status"`
	NewStatus           string `json:"new_status"`
	ChangedBy           string `json:"changed_by"`
	Timestamp           string `json:"timestamp"`
	EstimatedCompletion string `json:"estimated_completion"`
}

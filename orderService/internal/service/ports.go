package service

import "restaurant-system/orderService/internal/domain"

type OrderRepo interface {
	CreateOrderAdapter(req domain.CreateOrderRequest, total float64, priority int, number string) (domain.OrderResponce, error)
}

type RabbitRepo interface {
	PublishOrder(req domain.CreateOrderRequest, number string, priority int, total_amount float64) error
}

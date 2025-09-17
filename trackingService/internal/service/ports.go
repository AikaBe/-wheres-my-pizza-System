package service

import "restaurant-system/trackingService/internal/domain"

type TrackingRepo interface {
	GetOrderById(id string) (domain.TrackingService, error)
	GetHistory(id string) ([]domain.History, error)
	GetWorkers() ([]domain.Workers, error)
}

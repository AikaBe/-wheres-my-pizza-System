package domain

import "time"

type TrackingService struct {
	OrderNumber         string
	CurrentStatus       string
	UpdatedAt           time.Time
	EstimatedCompletion *time.Time
	ProcessedBy         string
}

type History struct {
	Status    string
	Timestamp time.Time
	ChangedBy string
}

type Workers struct {
	WorkerName      string
	Status          string
	OrdersProcessed int
	LastSeen        time.Time
}

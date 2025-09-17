package service

import "restaurant-system/trackingService/internal/domain"

type TrackerService struct {
	trackingRepo TrackingRepo
}

func NewTrackerService(trackingRepo TrackingRepo) *TrackerService {
	return &TrackerService{trackingRepo: trackingRepo}
}

func (s *TrackerService) GetOrderByIdService(id string) (domain.TrackingService, error) {
	return s.trackingRepo.GetOrderById(id)
}

func (s *TrackerService) GetHistoryService(id string) ([]domain.History, error) {
	return s.trackingRepo.GetHistory(id)
}

func (s *TrackerService) GetWorkers() ([]domain.Workers, error) {
	return s.trackingRepo.GetWorkers()
}

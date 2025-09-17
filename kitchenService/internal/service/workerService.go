package service

import (
	"context"
	"errors"
	"wheres-my-pizza/kitchenService/internal/adapter/postgre"
)

type WorkerService struct {
	repo *postgre.WorkerRepository
}

func NewWorkerService(repo *postgre.WorkerRepository) *WorkerService {
	return &WorkerService{repo: repo}
}

// 🔹 Регистрация воркера с проверкой дубликатов
func (s *WorkerService) RegisterWorker(ctx context.Context, name, wtype string) error {
	// проверяем, есть ли уже активный воркер с таким именем
	isActive, err := s.repo.IsWorkerActive(ctx, name)
	if err != nil {
		return err
	}

	if isActive {
		return errors.New("worker with this name is already active")
	}

	// регистрируем воркера
	return s.repo.RegisterWorker(ctx, name, wtype)
}

// 🔹 Отправка heartbeat
func (s *WorkerService) SendHeartbeat(ctx context.Context, name string) error {
	return s.repo.SendHeartbeat(ctx, name)
}

// 🔹 Пометить воркера как offline
func (s *WorkerService) MarkOffline(ctx context.Context, name string) error {
	return s.repo.UpdateWorkerStatus(ctx, name, "offline")
}

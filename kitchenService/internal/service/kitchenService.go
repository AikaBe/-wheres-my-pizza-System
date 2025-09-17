// internal/service/kitchen_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"restaurant-system/kitchenWorker/internal/domain"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type KitchenService struct {
	dbRepo         domain.KitchenRepository
	rabbitRepo     domain.RabbitMQConsumer
	workerName     string
	orderTypes     []string
	prefetch       int
	isShuttingDown bool
}

func NewKitchenService(
	dbRepo domain.KitchenRepository,
	rabbitRepo domain.RabbitMQConsumer,
	workerName string,
	orderTypes []string,
	prefetch int,
) *KitchenService {
	return &KitchenService{
		dbRepo:     dbRepo,
		rabbitRepo: rabbitRepo,
		workerName: workerName,
		orderTypes: orderTypes,
		prefetch:   prefetch,
	}
}

func (s *KitchenService) RegisterWorker(ctx context.Context) error {
	workerType := "general"
	if len(s.orderTypes) > 0 {
		workerType = strings.Join(s.orderTypes, ",")
	}

	if err := s.dbRepo.RegisterWorker(ctx, s.workerName, workerType); err != nil {
		slog.Error("Worker registration failed", "error", err, "worker", s.workerName)
		return err
	}

	slog.Info("Worker registered successfully", "worker", s.workerName, "type", workerType)
	return nil
}

func (s *KitchenService) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.dbRepo.UpdateWorkerHeartbeat(ctx, s.workerName); err != nil {
				slog.Error("Heartbeat failed", "error", err, "worker", s.workerName)
			} else {
				slog.Debug("Heartbeat sent", "worker", s.workerName)
			}
		}
	}
}

func (s *KitchenService) ConsumeOrders(ctx context.Context) {
	queueName := fmt.Sprintf("kitchen_queue_%s", s.workerName)

	err := s.rabbitRepo.ConsumeMessages(ctx, queueName, s.prefetch, func(msg amqp.Delivery) error {
		if s.isShuttingDown {
			msg.Nack(false, true) // Requeue for other workers
			return nil
		}

		return s.processOrderMessage(ctx, msg)
	})
	if err != nil {
		slog.Error("Failed to consume messages", "error", err)
	}
}

func (s *KitchenService) processOrderMessage(ctx context.Context, msg amqp.Delivery) error {
	var order domain.OrderMessage
	if err := json.Unmarshal(msg.Body, &order); err != nil {
		slog.Error("Failed to unmarshal order message", "error", err)
		msg.Nack(false, false) // Don't requeue malformed messages
		return err
	}

	// Check if worker can handle this order type
	if !s.canHandleOrderType(order.OrderType) {
		slog.Debug("Worker cannot handle order type", "order_type", order.OrderType, "worker", s.workerName)
		msg.Nack(false, true) // Requeue for appropriate worker
		return nil
	}

	// Check if order is already being processed
	currentStatus, err := s.dbRepo.GetOrderCurrentStatus(ctx, order.OrderNumber)
	if err != nil {
		slog.Error("Failed to get order status", "error", err, "order", order.OrderNumber)
		msg.Nack(false, true) // Requeue
		return err
	}

	// Idempotency check
	if currentStatus == "cooking" || currentStatus == "ready" {
		slog.Debug("Order already processed", "order", order.OrderNumber, "status", currentStatus)
		msg.Ack(false) // Acknowledge to remove from queue
		return nil
	}

	// Process order
	if err := s.processOrder(ctx, order); err != nil {
		slog.Error("Order processing failed", "error", err, "order", order.OrderNumber)
		msg.Nack(false, true) // Requeue for retry
		return err
	}

	msg.Ack(false) // Acknowledge successful processing
	slog.Debug("Order completed", "order", order.OrderNumber, "worker", s.workerName)
	return nil
}

func (s *KitchenService) canHandleOrderType(orderType string) bool {
	if len(s.orderTypes) == 0 {
		return true // General worker handles all types
	}

	for _, t := range s.orderTypes {
		if t == orderType {
			return true
		}
	}
	return false
}

func (s *KitchenService) processOrder(ctx context.Context, order domain.OrderMessage) error {
	// Set status to cooking
	if err := s.dbRepo.UpdateOrderStatus(ctx, order.OrderNumber, "cooking", s.workerName); err != nil {
		return err
	}

	// Publish status update
	estimatedCompletion := s.calculateEstimatedCompletion(order.OrderType)
	statusUpdate := domain.StatusUpdateMessage{
		OrderNumber:         order.OrderNumber,
		OldStatus:           "received",
		NewStatus:           "cooking",
		ChangedBy:           s.workerName,
		Timestamp:           time.Now().UTC(),
		EstimatedCompletion: estimatedCompletion,
	}

	if err := s.rabbitRepo.PublishStatusUpdate(ctx, statusUpdate); err != nil {
		slog.Error("Failed to publish status update", "error", err)
	}

	// Simulate cooking time
	cookingTime := s.getCookingTime(order.OrderType)
	time.Sleep(cookingTime)

	// Complete order
	if err := s.dbRepo.CompleteOrder(ctx, order.OrderNumber, s.workerName); err != nil {
		return err
	}

	// Publish ready status update
	readyUpdate := domain.StatusUpdateMessage{
		OrderNumber:         order.OrderNumber,
		OldStatus:           "cooking",
		NewStatus:           "ready",
		ChangedBy:           s.workerName,
		Timestamp:           time.Now().UTC(),
		EstimatedCompletion: time.Now().UTC(),
	}

	return s.rabbitRepo.PublishStatusUpdate(ctx, readyUpdate)
}

func (s *KitchenService) getCookingTime(orderType string) time.Duration {
	switch orderType {
	case "dine_in":
		return 8 * time.Second
	case "takeout":
		return 10 * time.Second
	case "delivery":
		return 12 * time.Second
	default:
		return 10 * time.Second
	}
}

func (s *KitchenService) calculateEstimatedCompletion(orderType string) time.Time {
	cookingTime := s.getCookingTime(orderType)
	return time.Now().UTC().Add(cookingTime)
}

func (s *KitchenService) Shutdown(ctx context.Context) {
	s.isShuttingDown = true

	// Set worker offline
	if err := s.dbRepo.SetWorkerOffline(ctx, s.workerName); err != nil {
		slog.Error("Failed to set worker offline", "error", err)
	}

	// Close RabbitMQ connection
	if err := s.rabbitRepo.Close(); err != nil {
		slog.Error("Failed to close RabbitMQ connection", "error", err)
	}

	slog.Info("Worker shutdown complete", "worker", s.workerName)
}

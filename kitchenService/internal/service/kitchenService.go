package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"restaurant-system/kitchenService/internal/domain"
	"restaurant-system/logger"

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
	workerType := strings.Join(s.orderTypes, ",")

	if err := s.dbRepo.RegisterWorker(ctx, s.workerName, workerType); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "register-worker", "Worker registration failed", err)
		return err
	}

	logger.Log(logger.INFO, "kitchen-worker", "register-worker", "Worker registered successfully", nil)
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
				logger.Log(logger.ERROR, "kitchen-worker", "heartbeat", "Heartbeat failed", err)
			} else {
				logger.Log(logger.DEBUG, "kitchen-worker", "heartbeat", "Heartbeat sent", nil)
			}
		}
	}
}

func (s *KitchenService) ConsumeOrders(ctx context.Context) {
	queueName := fmt.Sprintf("kitchen_queue_%s", s.workerName)

	err := s.rabbitRepo.ConsumeMessages(ctx, queueName, s.prefetch, func(msg amqp.Delivery) error {
		if s.isShuttingDown {
			msg.Nack(false, true)
			return nil
		}
		return s.processOrderMessage(ctx, msg)
	})
	if err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "consume-orders", "Failed to consume messages", err)
	}
}

func (s *KitchenService) processOrderMessage(ctx context.Context, msg amqp.Delivery) error {
	var order domain.OrderMessage
	if err := json.Unmarshal(msg.Body, &order); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "process-order", "Failed to unmarshal order message", err)
		msg.Nack(false, false)
		return err
	}

	if !s.canHandleOrderType(order.OrderType) {
		logger.Log(logger.DEBUG, "kitchen-worker", "process-order", "Worker cannot handle order type", nil)
		msg.Nack(false, true)
		return nil
	}

	currentStatus, err := s.dbRepo.GetOrderCurrentStatus(ctx, order.OrderNumber)
	if err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "process-order", "Failed to get order status", err)
		msg.Nack(false, true)
		return err
	}

	if currentStatus == "cooking" || currentStatus == "ready" {
		logger.Log(logger.DEBUG, "kitchen-worker", "process-order", fmt.Sprintf("Order already processed, status=%s", currentStatus), nil)
		msg.Ack(false)
		return nil
	}

	if err := s.processOrder(ctx, order); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "process-order", "Order processing failed", err)
		msg.Nack(false, true)
		return err
	}

	msg.Ack(false)
	logger.Log(logger.DEBUG, "kitchen-worker", "process-order", "Order completed", nil)
	return nil
}

func (s *KitchenService) canHandleOrderType(orderType string) bool {
	if len(s.orderTypes) == 0 {
		return true
	}
	for _, t := range s.orderTypes {
		if t == orderType {
			return true
		}
	}
	return false
}

func (s *KitchenService) processOrder(ctx context.Context, order domain.OrderMessage) error {
	if err := s.dbRepo.UpdateOrderStatus(ctx, order.OrderNumber, "cooking", s.workerName); err != nil {
		return err
	}
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
		logger.Log(logger.ERROR, "kitchen-worker", "publish-status", "Failed to publish status update", err)
	}

	cookingTime := s.getCookingTime(order.OrderType)
	time.Sleep(cookingTime)

	if err := s.dbRepo.CompleteOrder(ctx, order.OrderNumber, s.workerName); err != nil {
		return err
	}
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

	if err := s.dbRepo.SetWorkerOffline(ctx, s.workerName); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "shutdown", "Failed to set worker offline", err)
	}

	if err := s.rabbitRepo.Close(); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "shutdown", "Failed to close RabbitMQ connection", err)
	}

	logger.Log(logger.INFO, "kitchen-worker", "shutdown", "Worker shutdown complete", nil)
}

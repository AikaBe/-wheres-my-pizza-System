package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"wheres-my-pizza/kitchenService/internal/adapter/postgre"
	"wheres-my-pizza/kitchenService/internal/adapter/rabbitMq"
	"wheres-my-pizza/kitchenService/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type KitchenService struct {
	repo              *postgre.KitchenRepo
	rabbit            *rabbitMq.RabbitMq
	workerName        string
	orderTypes        []string
	notificationsChan *amqp.Channel
}

func NewKitchenService(repo *postgre.KitchenRepo, rabbit *rabbitMq.RabbitMq, name string, orderTypes []string) *KitchenService {
	// создаем канал для нотификаций
	ch, err := rabbit.GetConnection().Channel()
	if err != nil {
		slog.Error("Failed to create notification channel", "error", err)
		return nil
	}

	return &KitchenService{
		repo:              repo,
		rabbit:            rabbit,
		workerName:        name,
		orderTypes:        orderTypes,
		notificationsChan: ch,
	}
}

// 🔹 Закрытие ресурсов
func (s *KitchenService) Close() {
	if s.notificationsChan != nil {
		s.notificationsChan.Close()
	}
}

// 🔹 Обработка сообщений
func (s *KitchenService) ProcessMessages(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			s.processOrder(ctx, msg)
		}
	}
}

func (s *KitchenService) processOrder(ctx context.Context, msg amqp.Delivery) {
	var order domain.OrderMessage
	if err := json.Unmarshal(msg.Body, &order); err != nil {
		slog.Error("Invalid message", "error", err)
		s.rabbit.Reject(msg) // невалидное сообщение → в DLQ
		return
	}

	slog.Debug("Order processing started", "order_id", order.OrderID, "order_type", order.OrderType)

	// проверка специализации воркера
	if len(s.orderTypes) > 0 && !s.canHandleOrderType(order.OrderType) {
		slog.Debug("Worker not specialized for this order type", "order_type", order.OrderType, "worker_types", s.orderTypes)
		s.rabbit.Nack(msg) // возвращаем в очередь для другого воркера
		return
	}

	// проверяем, не обрабатывается ли уже заказ (идемпотентность)
	currentStatus, err := s.repo.GetOrderStatus(ctx, order.OrderID)
	if err != nil {
		slog.Error("Failed to get order status", "error", err)
		s.rabbit.Nack(msg) // временная ошибка → вернуть в очередь
		return
	}

	if currentStatus == "cooking" || currentStatus == "ready" {
		slog.Debug("Order already processed", "order_id", order.OrderID, "status", currentStatus)
		s.rabbit.Ack(msg) // уже обработан → подтверждаем
		return
	}

	// начинаем обработку заказа
	if err := s.repo.StartCookingOrder(ctx, order.OrderID, s.workerName); err != nil {
		slog.Error("Failed to start cooking order", "error", err)
		s.rabbit.Nack(msg) // временная ошибка → вернуть в очередь
		return
	}

	// публикуем уведомление о начале готовки
	s.publishStatusUpdate(order.OrderID, "received", "cooking", order.OrderType)

	// симуляция готовки
	select {
	case <-time.After(s.simulateCooking(order.OrderType)):
		// продолжить обработку
	case <-ctx.Done():
		slog.Info("Order processing cancelled", "order_id", order.OrderID)
		s.rabbit.Nack(msg) // возвращаем в очередь при отмене
		return
	}

	// завершаем заказ
	if err := s.repo.CompleteOrder(ctx, order.OrderID, s.workerName); err != nil {
		slog.Error("Failed to complete order", "error", err)
		s.rabbit.Nack(msg) // временная ошибка → вернуть в очередь
		return
	}

	// публикуем уведомление о готовности
	s.publishStatusUpdate(order.OrderID, "cooking", "ready", order.OrderType)

	slog.Debug("Order completed", "order_id", order.OrderID)
	s.rabbit.Ack(msg) // подтверждаем обработку
}

// 🔹 Проверка возможности обработки типа заказа
func (s *KitchenService) canHandleOrderType(orderType string) bool {
	for _, t := range s.orderTypes {
		if strings.EqualFold(t, orderType) {
			return true
		}
	}
	return false
}

// 🔹 Публикация уведомления о смене статуса
func (s *KitchenService) publishStatusUpdate(orderID string, oldStatus, newStatus, orderType string) {
	notification := domain.StatusNotification{
		OrderNumber:         orderID,
		OldStatus:           oldStatus,
		NewStatus:           newStatus,
		ChangedBy:           s.workerName,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
		EstimatedCompletion: time.Now().UTC().Add(s.simulateCooking(orderType)).Format(time.RFC3339),
	}

	body, err := json.Marshal(notification)
	if err != nil {
		slog.Error("Failed to marshal notification", "error", err)
		return
	}

	err = s.notificationsChan.Publish(
		"notifications_fanout", // exchange
		"",                     // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		slog.Error("Failed to publish notification", "error", err)
	}
}

// 🔹 Симуляция готовки
func (s *KitchenService) simulateCooking(orderType string) time.Duration {
	switch orderType {
	case "dine_in":
		return 8 * time.Second
	case "takeout":
		return 10 * time.Second
	case "delivery":
		return 12 * time.Second
	default:
		return 5 * time.Second
	}
}

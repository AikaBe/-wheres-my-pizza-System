package notification

import (
	"restaurant-system/logger"
	"restaurant-system/notificationService/rabbitMq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NotificationMain() error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Log(logger.ERROR, "notification-service", "init-mq", "Unable to connect to RabbitMQ", err)
		return err
	}
	logger.Log(logger.INFO, "notification-service", "init-mq", "Connected to RabbitMQ", nil)
	defer conn.Close()

	rabbitRepo, err := rabbitMq.NewRabbitMq(conn)
	if err != nil {
		logger.Log(logger.ERROR, "notification-service", "init-rabbit-repo", "Unable to init RabbitMQ repo", err)
		return err
	}
	defer rabbitRepo.Close()

	msgs, err := rabbitRepo.Consume("notifications_queue")
	if err != nil {
		logger.Log(logger.ERROR, "notification-service", "consume", "Unable to consume messages", err)
		return err
	}

	logger.Log(logger.INFO, "notification-service", "startup", "Notification Service started, waiting for messages...", nil)

	for msg := range msgs {
		logger.Log(logger.INFO, "notification-service", "notification-received", "Order received: "+string(msg.Body), nil)
		msg.Ack(false)
	}

	return nil
}

// internal/adapter/rabbitmq/consumer.go
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"restaurant-system/kitchenWorker/internal/domain"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQConsumer(conn *amqp.Connection) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		"orders_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
	}, nil
}

func (r *RabbitMQConsumer) ConsumeMessages(ctx context.Context, queueName string, prefetch int, handler func(msg amqp.Delivery) error) error {
	// Set prefetch
	err := r.channel.Qos(prefetch, 0, false)
	if err != nil {
		return err
	}

	// Declare queue
	q, err := r.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Bind to all kitchen routes
	err = r.channel.QueueBind(
		q.Name,
		"kitchen.*.*",
		"orders_topic",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := r.channel.Consume(
		q.Name,
		"",
		false, // auto-ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}
			if err := handler(msg); err != nil {
				// Handle error (nack with requeue or send to DLQ)
				msg.Nack(false, true) // requeue
			}
		}
	}
}

func (r *RabbitMQConsumer) PublishStatusUpdate(ctx context.Context, update domain.StatusUpdateMessage) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.channel.PublishWithContext(
		ctx,
		"notifications_fanout",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (r *RabbitMQConsumer) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	return r.conn.Close()
}

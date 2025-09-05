package rabbitMq

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
	"wheres-my-pizza/orderService/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMq struct {
	rabbitMq *amqp.Connection
	channel  *amqp.Channel
}

type OrderMessage struct {
	OrderNumber     string                   `json:"order_number"`
	CustomerName    string                   `json:"customer_name"`
	OrderType       string                   `json:"order_type"`
	TableNumber     int                      `json:"table_number,omitempty"`
	DeliveryAddress string                   `json:"delivery_address,omitempty"`
	Items           []domain.CreateOrderItem `json:"items"`
	TotalAmount     float64                  `json:"total_amount"`
	Priority        int                      `json:"priority"`
}

func NewRabbitMq(conn *amqp.Connection) (*RabbitMq, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

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

	q, err := ch.QueueDeclare(
		"kitchen_orders",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = ch.QueueBind(
		q.Name,
		"kitchen.*.*",
		"orders_topic",
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMq{
		rabbitMq: conn,
		channel:  ch,
	}, nil
}

func (r *RabbitMq) PublishOrder(req domain.CreateOrderRequest, number string, priority int, total_amount float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := OrderMessage{
		OrderNumber:     number,
		CustomerName:    req.CustomerName,
		OrderType:       req.Type,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		Items:           req.Items,
		TotalAmount:     total_amount,
		Priority:        priority,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	pr := strconv.Itoa(priority)
	routingKey := "kitchen." + req.Type + "." + pr

	return r.channel.PublishWithContext(
		ctx,
		"orders_topic",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

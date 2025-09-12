package rabbitMq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMq struct {
	rabbitConn *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitMq(conn *amqp.Connection) (*RabbitMq, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = channel.ExchangeDeclare(
		"notifications_fanout",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	q, err := channel.QueueDeclare(
		"notifications_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = channel.QueueBind(
		q.Name,
		"",
		"notifications_fanout",
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMq{
		rabbitConn: conn,
		channel:    channel,
	}, nil
}

func (r *RabbitMq) Consume(queue string) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

func (r *RabbitMq) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.rabbitConn != nil {
		r.rabbitConn.Close()
	}
}

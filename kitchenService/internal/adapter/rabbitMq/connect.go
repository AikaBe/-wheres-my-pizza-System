package rabbitMq

import (
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMq struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

// Подключение к RabbitMQ и настройка очереди
func NewRabbitMq(conn *amqp.Connection, queueName string, prefetch int) (*RabbitMq, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Prefetch limit (basic.qos), чтобы ограничить кол-во сообщений на одного воркера
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}

	// Создаем DLQ
	_, err = ch.QueueDeclare(
		queueName+"_dlq",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Аргументы для основной очереди с DLQ
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": queueName + "_dlq",
	}

	// Очередь для кухни
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,
	)
	if err != nil {
		return nil, err
	}

	// биндим очередь на exchange orders_topic
	err = ch.QueueBind(
		q.Name,
		"kitchen.*.*", // принимает все типы заказов (можно фильтровать по type/prio)
		"orders_topic",
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMq{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

// Подписка на очередь
func (r *RabbitMq) Consume() (<-chan amqp.Delivery, error) {
	msgs, err := r.channel.Consume(
		r.queue.Name,
		"",
		false, // autoAck = false → вручную ack/nack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	slog.Info("Listening RabbitMQ queue", "queue", r.queue.Name)
	return msgs, nil
}

// ACK подтверждение
func (r *RabbitMq) Ack(msg amqp.Delivery) error {
	return msg.Ack(false)
}

// NACK с возвратом в очередь
func (r *RabbitMq) Nack(msg amqp.Delivery) error {
	return msg.Nack(false, true)
}

// Reject с отправкой в DLQ
func (r *RabbitMq) Reject(msg amqp.Delivery) error {
	return msg.Reject(false)
}

// 🔹 Получение соединения
func (r *RabbitMq) GetConnection() *amqp.Connection {
	return r.conn
}

// 🔹 Публикация сообщения
func (r *RabbitMq) Publish(exchange, routingKey string, body []byte) error {
	return r.channel.Publish(
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// 🔹 Закрытие соединений
func (r *RabbitMq) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

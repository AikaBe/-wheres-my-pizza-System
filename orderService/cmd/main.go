package cmd

import (
	"context"
	"net/http"
	"os"
	"restaurant-system/logger"
	"restaurant-system/orderService/internal/adapter/postgre"
	"restaurant-system/orderService/internal/adapter/rabbitMq"
	"restaurant-system/orderService/internal/handler"
	"restaurant-system/orderService/internal/service"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

func MainOrder(port string, maxConcurrent int) {
	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		logger.Log(logger.ERROR, "order-service", "init-db", "Unable to connect to database", err)
		os.Exit(1)
	}
	logger.Log(logger.INFO, "order-service", "init-db", "Connected to database", nil)
	defer conn.Close(context.Background())

	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Log(logger.ERROR, "order-service", "init-mq", "Unable to connect to RabbitMQ", err)
		os.Exit(1)
	}
	logger.Log(logger.INFO, "order-service", "init-mq", "Connected to RabbitMQ", nil)
	defer connMq.Close()

	rabbitRepo, err := rabbitMq.NewRabbitMq(connMq)
	if err != nil {
		logger.Log(logger.ERROR, "order-service", "init-rabbit-repo", "Unable to init RabbitMQ repo", err)
		os.Exit(1)
	}

	orderRepo := postgre.NewOrderRepo(conn)
	orderService := service.NewOrderService(orderRepo, rabbitRepo)
	orderHandler := handler.NewOrderHandler(orderService, maxConcurrent)

	http.HandleFunc("/orders", orderHandler.CreateOrderHandler)

	logger.Log(logger.INFO, "order-service", "startup",
		"Order Service started on port "+port+" with max_concurrent="+
			string(rune(maxConcurrent)), nil)

	addr := ":" + port
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Log(logger.ERROR, "order-service", "http-server", "Unable to start server", err)
		os.Exit(1)
	}
}

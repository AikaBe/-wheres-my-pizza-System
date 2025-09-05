package cmd

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"wheres-my-pizza/orderService/internal/adapter/postgre"
	"wheres-my-pizza/orderService/internal/adapter/rabbitMq"
	"wheres-my-pizza/orderService/internal/handler"
	"wheres-my-pizza/orderService/internal/service"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

func MainOrder(port string, maxConcurrent int) {
	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to database")
	defer conn.Close(context.Background())

	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Unable to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to RabbitMQ")
	defer connMq.Close()

	rabbitRepo, err := rabbitMq.NewRabbitMq(connMq)
	if err != nil {
		slog.Error("Unable to init RabbitMQ repo", "error", err)
		os.Exit(1)
	}

	orderRepo := postgre.NewOrderRepo(conn)
	orderService := service.NewOrderService(orderRepo, rabbitRepo)
	orderHandler := handler.NewOrderHandler(orderService, maxConcurrent)

	http.HandleFunc("/order", orderHandler.CreateOrderHandler)

	slog.Info("Order Service started",
		"port", port,
		"max_concurrent", maxConcurrent,
	)

	addr := ":" + port
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Unable to start server", "error", err)
		os.Exit(1)
	}
}

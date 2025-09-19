package kitchenService

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"restaurant-system/kitchenService/internal/adapter/postgre"
	"restaurant-system/kitchenService/internal/adapter/rabbitmq"
	"restaurant-system/kitchenService/internal/service"
	"restaurant-system/logger"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

func MainKitchenWorker(workerName string, orderTypes []string, heartbeatInterval, prefetch int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "init-db", "Unable to connect to database", err)
		os.Exit(1)
	}
	logger.Log(logger.INFO, "kitchen-worker", "init-db", "Connected to database", nil)
	defer conn.Close(context.Background())

	// RabbitMQ connection
	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Unable to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer connMq.Close()
	slog.Info("Connected to RabbitMQ")

	// Initialize repositories
	dbRepo := postgre.NewKitchenRepo(conn)
	rabbitRepo, err := rabbitmq.NewRabbitMQConsumer(connMq)
	if err != nil {
		slog.Error("Unable to init RabbitMQ consumer", "error", err)
		os.Exit(1)
	}

	// Initialize service
	kitchenSrv := service.NewKitchenService(dbRepo, rabbitRepo, workerName, orderTypes, prefetch)

	// Register worker
	if err := kitchenSrv.RegisterWorker(ctx); err != nil {
		slog.Error("Worker registration failed", "error", err)
		os.Exit(1)
	}

	// Start heartbeat
	go kitchenSrv.StartHeartbeat(ctx, time.Duration(heartbeatInterval)*time.Second)

	// Start consuming messages
	go kitchenSrv.ConsumeOrders(ctx)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Starting graceful shutdown...")
	kitchenSrv.Shutdown(ctx)
	slog.Info("Kitchen Worker shutdown complete")
}

package kitchenService

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"restaurant-system/kitchenService/internal/adapter/postgre"
	"restaurant-system/kitchenService/internal/adapter/rabbitmq"
	"restaurant-system/kitchenService/internal/service"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

func MainKitchenWorker(workerName string, orderTypes []string, heartbeatInterval, prefetch int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database connection
	dbPool, err := pgxpool.New(ctx, "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable")
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	slog.Info("Connected to database")

	// RabbitMQ connection
	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Unable to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer connMq.Close()
	slog.Info("Connected to RabbitMQ")

	// Initialize repositories
	dbRepo := postgre.NewKitchenRepo(dbPool)
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

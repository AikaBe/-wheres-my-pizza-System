// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"restaurant-system/kitchenWorker/internal/adapter/postgre"
	"restaurant-system/kitchenWorker/internal/adapter/rabbitmq"
	"restaurant-system/kitchenWorker/internal/service"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	mode := flag.String("mode", "", "Service mode")
	workerName := flag.String("worker-name", "", "Worker name")
	orderTypes := flag.String("order-types", "", "Order types (comma-separated)")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Heartbeat interval in seconds")
	prefetch := flag.Int("prefetch", 1, "Prefetch count")
	flag.Parse()

	if *mode != "kitchen-worker" {
		slog.Error("Invalid mode", "mode", *mode)
		os.Exit(1)
	}

	if *workerName == "" {
		slog.Error("Worker name is required")
		os.Exit(1)
	}

	MainKitchenWorker(*workerName, *orderTypes, *heartbeatInterval, *prefetch)
}

func MainKitchenWorker(workerName, orderTypes string, heartbeatInterval, prefetch int) {
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

	// Parse order types
	var orderTypesList []string
	if orderTypes != "" {
		orderTypesList = splitOrderTypes(orderTypes)
	}

	// Initialize service
	kitchenService := service.NewKitchenService(dbRepo, rabbitRepo, workerName, orderTypesList, prefetch)

	// Register worker
	if err := kitchenService.RegisterWorker(ctx); err != nil {
		slog.Error("Worker registration failed", "error", err)
		os.Exit(1)
	}

	// Start heartbeat
	go kitchenService.StartHeartbeat(ctx, time.Duration(heartbeatInterval)*time.Second)

	// Start consuming messages
	go kitchenService.ConsumeOrders(ctx)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Starting graceful shutdown...")
	kitchenService.Shutdown(ctx)
	slog.Info("Kitchen Worker shutdown complete")
}

func splitOrderTypes(orderTypes string) []string {
	var result []string
	// Simple split implementation
	// In real code would handle trimming and validation
	return result
}

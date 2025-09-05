package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wheres-my-pizza/kitchenWorker/internal/adapter/postgre"
	"wheres-my-pizza/kitchenWorker/internal/adapter/rabbitMq"
	"wheres-my-pizza/kitchenWorker/internal/service"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

func MainKitchenWorker(workerName string, orderTypes []string, heartbeatInterval, prefetch int) {
	// --- Connect DB ---
	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to database")
	defer conn.Close(context.Background())

	// --- Connect RabbitMQ ---
	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Unable to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to RabbitMQ")
	defer connMq.Close()

	rabbitRepo, err := rabbitMq.NewRabbitMq(connMq, prefetch)
	if err != nil {
		slog.Error("Unable to init RabbitMQ repo", "error", err)
		os.Exit(1)
	}

	workerRepo := postgre.NewWorkerRepo(conn)
	orderRepo := postgre.NewOrderRepo(conn)

	// --- Init Service ---
	kitchenService := service.NewKitchenService(workerRepo, orderRepo, rabbitRepo, workerName, orderTypes)

	// --- Worker Registration ---
	if err := kitchenService.RegisterWorker(context.Background()); err != nil {
		slog.Error("worker_registration_failed", "error", err)
		os.Exit(1)
	}
	slog.Info("worker_registered", "name", workerName)

	// --- Heartbeat goroutine ---
	go func() {
		ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := kitchenService.Heartbeat(context.Background()); err != nil {
				slog.Error("heartbeat_failed", "error", err)
			} else {
				slog.Debug("heartbeat_sent", "worker", workerName)
			}
		}
	}()

	// --- Message Consumer ---
	go func() {
		if err := kitchenService.ConsumeOrders(context.Background()); err != nil {
			slog.Error("message_processing_failed", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful Shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	slog.Info("graceful_shutdown", "worker", workerName)

	if err := kitchenService.Shutdown(context.Background()); err != nil {
		slog.Error("failed_to_shutdown_worker", "error", err)
	}

	os.Exit(0)
}

package kitchenService

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"restaurant-system/kitchenService/internal/adapter/postgre"
	"restaurant-system/kitchenService/internal/adapter/rabbitmq"
	"restaurant-system/kitchenService/internal/service"
	"restaurant-system/logger"

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

	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "init-rabbitmq", "Unable to connect to RabbitMQ", err)
		os.Exit(1)
	}
	defer connMq.Close()
	logger.Log(logger.INFO, "kitchen-worker", "init-rabbitmq", "Connected to RabbitMQ", nil)

	dbRepo := postgre.NewKitchenRepo(conn)
	rabbitRepo, err := rabbitmq.NewRabbitMQConsumer(connMq)
	if err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "init-rabbitmq-consumer", "Unable to init RabbitMQ consumer", err)
		os.Exit(1)
	}

	kitchenSrv := service.NewKitchenService(dbRepo, rabbitRepo, workerName, orderTypes, prefetch)

	if err := kitchenSrv.RegisterWorker(ctx); err != nil {
		logger.Log(logger.ERROR, "kitchen-worker", "register-worker", "Worker registration failed", err)
		os.Exit(1)
	}

	go kitchenSrv.StartHeartbeat(ctx, time.Duration(heartbeatInterval)*time.Second)
	go kitchenSrv.ConsumeOrders(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Log(logger.INFO, "kitchen-worker", "shutdown", "Starting graceful shutdown...", nil)
	kitchenSrv.Shutdown(ctx)
	logger.Log(logger.INFO, "kitchen-worker", "shutdown", "Kitchen Worker shutdown complete", nil)
}

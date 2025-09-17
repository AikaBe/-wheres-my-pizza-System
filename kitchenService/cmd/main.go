package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"wheres-my-pizza/kitchenService/internal/adapter/postgre"
	"wheres-my-pizza/kitchenService/internal/adapter/rabbitMq"
	"wheres-my-pizza/kitchenService/internal/service"
)

func MainKitchen(workerName string, orderTypes []string, prefetch int, heartbeatInterval int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// подключение к PostgreSQL с использованием пула
	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to database")
	defer pool.Close()

	// подключение к RabbitMQ
	connMq, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Unable to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to RabbitMQ")
	defer connMq.Close()

	// инициализация репозиториев
	workerRepo := postgre.NewWorkerRepository(pool)
	kitchenRepo := postgre.NewKitchenRepo(pool)

	// сервис регистрации воркера
	workerService := service.NewWorkerService(workerRepo)

	// регистрация воркера с проверкой дубликатов
	workerType := "general"
	if len(orderTypes) > 0 {
		workerType = strings.Join(orderTypes, ",")
	}

	if err := workerService.RegisterWorker(ctx, workerName, workerType); err != nil {
		slog.Error("Worker registration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Worker registered", "name", workerName, "type", workerType)

	// инициализация RabbitMQ адаптера
	rabbit, err := rabbitMq.NewRabbitMq(connMq, "kitchen_queue", prefetch)
	if err != nil {
		slog.Error("Unable to init RabbitMQ repo", "error", err)
		os.Exit(1)
	}
	defer rabbit.Close()

	// инициализация сервиса кухни
	kitchenService := service.NewKitchenService(kitchenRepo, rabbit, workerName, orderTypes)
	if kitchenService == nil {
		slog.Error("Failed to create kitchen service")
		os.Exit(1)
	}
	defer kitchenService.Close()

	// запуск heartbeat
	go func() {
		ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if err := workerService.SendHeartbeat(ctx, workerName); err != nil {
					slog.Error("Heartbeat failed", "error", err)
					// При критической ошибке heartbeat завершаем работу
					cancel()
					return
				} else {
					slog.Debug("Heartbeat sent")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// запуск обработки сообщений
	msgChan, err := rabbit.Consume()
	if err != nil {
		slog.Error("Failed to start consuming", "error", err)
		os.Exit(1)
	}

	// обработка сообщений
	go kitchenService.ProcessMessages(ctx, msgChan)

	slog.Info("Kitchen worker started successfully", "worker", workerName, "types", orderTypes)

	// ожидание сигнала завершения
	sig := <-sigs
	slog.Info("Starting graceful shutdown...", "signal", sig)

	// завершаем обработку
	cancel()

	// небольшая задержка для завершения текущей обработки
	time.Sleep(2 * time.Second)

	// помечаем воркера как offline
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := workerService.MarkOffline(ctxShutdown, workerName); err != nil {
		slog.Error("Failed to mark worker offline", "error", err)
	}

	slog.Info("Kitchen worker stopped gracefully")
}
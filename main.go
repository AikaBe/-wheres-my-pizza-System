package main

import (
	"flag"
	"os"
	"restaurant-system/kitchenService"
	"restaurant-system/logger"
	"restaurant-system/notificationService/notification"
	"restaurant-system/orderService/cmd"
	"restaurant-system/trackingService"
	"strings"
)

func main() {
	mode := flag.String("mode", "", "Which mode to use")
	port := flag.String("port", "", "Port to listen on")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process")
	workerName := flag.String("worker-name", "", "Name of the worker")
	orderTypes := flag.String("order-types", "", "Comma-separated list of order types the worker can handle (e.g., dine_in,takeout,delivery)")
	prefetch := flag.Int("prefetch", 1, "RabbitMQ prefetch count, limiting how many messages the worker receives at once")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Interval in seconds between heartbeats")

	flag.Parse()

	switch *mode {
	case "order-service":
		cmd.MainOrder(*port, *maxConcurrent)
	case "notification-subscriber":
		if err := notification.NotificationMain(); err != nil {
			logger.Log(logger.ERROR,
				"notification-service",
				"startup",
				"Notification service failed",
				err,
			)
			os.Exit(1)
		}
	case "tracking-service":
		trackingService.TrackingMain(*port)
	case "kitchen-worker":
		// Валидация обязательных параметров для kitchen-worker
		if *workerName == "" {
			logger.Log(logger.ERROR, "kitchen-worker", "startup", "Worker name is required", nil)
			os.Exit(1)
		}

		// Парсинг типов заказов
		var orderTypesSlice []string
		if *orderTypes != "" {
			orderTypesSlice = strings.Split(*orderTypes, ",")
			// Убираем пробелы и приводим к нижнему регистру
			for i, ot := range orderTypesSlice {
				orderTypesSlice[i] = strings.TrimSpace(strings.ToLower(ot))
			}
		}

		// Валидация prefetch
		if *prefetch < 1 {
			logger.Log(logger.WARN, "kitchen-worker", "config", "Prefetch must be at least 1, using default value 1", nil)
			*prefetch = 1
		}

		// Валидация heartbeat interval
		if *heartbeatInterval < 5 {
			logger.Log(logger.WARN, "kitchen-worker", "config", "Heartbeat interval too low, using minimum 5 seconds", nil)
			*heartbeatInterval = 5
		}

		logger.Log(logger.INFO, "kitchen-worker", "startup", "Starting kitchen worker", map[string]interface{}{
			"worker_name":        *workerName,
			"order_types":        orderTypesSlice,
			"prefetch":           *prefetch,
			"heartbeat_interval": *heartbeatInterval,
		})

		kitchenService.MainKitchen(*workerName, orderTypesSlice, *prefetch, *heartbeatInterval)

	default:
		logger.Log(logger.ERROR, "main", "startup", "Unknown mode specified. Use: order-service, notification-subscriber, tracking-service, or kitchen-worker", map[string]interface{}{
			"mode": *mode,
		})
		os.Exit(1)
	}
}

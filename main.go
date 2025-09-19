package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"restaurant-system/kitchenService"
	"restaurant-system/logger"
	"restaurant-system/notificationService/notification"
	"restaurant-system/orderService/cmd"
	"restaurant-system/trackingService"
)

// Config для параметров kitchen-worker
type Config struct {
	WorkerName        string
	OrderTypes        []string
	Prefetch          int
	HeartbeatInterval int
}

func main() {
	mode := flag.String("mode", "", "Which mode to use")
	port := flag.String("port", "", "Port to listen on")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process")
	workerName := flag.String("worker-name", "", "Name of the worker")
	orderTypes := flag.String("order-types", "", "Comma-separated list of order types the worker can handle (e.g., dine_in,takeout,delivery)")
	prefetch := flag.Int("prefetch", 1, "RabbitMQ prefetch count")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Interval in seconds between heartbeats")
	flag.Parse()

	switch *mode {
	case "order-service":
		cmd.MainOrder(*port, *maxConcurrent)

	case "notification-subscriber":
		if err := notification.NotificationMain(); err != nil {
			logger.Log(logger.ERROR, "notification-service", "startup", "Notification service failed", err)
			os.Exit(1)
		}

	case "tracking-service":
		trackingService.TrackingMain(*port)

	case "kitchen-worker":
		cfg, valid := validateKitchenConfig(*workerName, *orderTypes, *prefetch, *heartbeatInterval)
		if !valid {
			os.Exit(1)
		}

		logger.Log(
			logger.INFO,
			"kitchen-worker",
			"startup",
			fmt.Sprintf(
				"Starting kitchen worker (name=%s, order_types=%v, prefetch=%d, heartbeat_interval=%d)",
				cfg.WorkerName, cfg.OrderTypes, cfg.Prefetch, cfg.HeartbeatInterval,
			),
			nil,
		)

		kitchenService.MainKitchenWorker(cfg.WorkerName, cfg.OrderTypes, cfg.Prefetch, cfg.HeartbeatInterval)

	default:
		logger.Log(
			logger.ERROR,
			"main",
			"startup",
			"Unknown mode specified. Use: order-service, notification-subscriber, tracking-service, or kitchen-worker",
			fmt.Errorf("mode=%s", *mode),
		)
		os.Exit(1)
	}
}

// validateKitchenConfig проверяет и нормализует параметры для kitchen-worker
func validateKitchenConfig(workerName, orderTypes string, prefetch, heartbeat int) (Config, bool) {
	if workerName == "" {
		logger.Log(logger.ERROR, "kitchen-worker", "startup", "Worker name is required", nil)
		return Config{}, false
	}

	var orderTypesSlice []string
	if orderTypes != "" {
		for _, ot := range strings.Split(orderTypes, ",") {
			ot = strings.TrimSpace(strings.ToLower(ot))
			if ot != "" {
				orderTypesSlice = append(orderTypesSlice, ot)
			}
		}
	}

	if prefetch < 1 {
		logger.Log(logger.ERROR, "kitchen-worker", "config", "Prefetch must be >= 1, setting to default=1", nil)
		prefetch = 1
	}

	if heartbeat < 5 {
		logger.Log(logger.ERROR, "kitchen-worker", "config", "Heartbeat must be >= 5, setting to default=5", nil)
		heartbeat = 5
	}

	return Config{
		WorkerName:        workerName,
		OrderTypes:        orderTypesSlice,
		Prefetch:          prefetch,
		HeartbeatInterval: heartbeat,
	}, true
}

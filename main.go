package main

import (
	"flag"
	"os"
	"restaurant-system/logger"
	"restaurant-system/notificationService/notification"
	"restaurant-system/orderService/cmd"
	"restaurant-system/trackingService"
)

func main() {
	mode := flag.String("mode", "", "Which mode to use")
	port := flag.String("port", "", "Port to listen on")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process")
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
	default:
		logger.Log(logger.ERROR, "tracking-service", "init-db", "Unable to connect to database", nil)
	}

}

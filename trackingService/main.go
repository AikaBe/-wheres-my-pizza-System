package trackingService

import (
	"context"
	"net/http"
	"os"
	"restaurant-system/logger"
	"restaurant-system/trackingService/internal/adapter"
	"restaurant-system/trackingService/internal/handler"
	"restaurant-system/trackingService/internal/service"

	"github.com/jackc/pgx/v5"
)

func TrackingMain(port string) {
	connStr := "postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		logger.Log(logger.ERROR, "tracking-service", "init-db", "Unable to connect to database", err)
		os.Exit(1)
	}
	logger.Log(logger.INFO, "tracking-service", "init-db", "Connected to database", nil)
	defer conn.Close(context.Background())

	trackerRepo := adapter.NewTrackerRepo(conn)
	trackerService := service.NewTrackerService(trackerRepo)
	trackerHandler := handler.NewTrackerHandler(trackerService)

	http.HandleFunc("/orders/", trackerHandler.GetTrackerHandler)
	http.HandleFunc("/workers/", trackerHandler.GetWorkers)

	logger.Log(logger.INFO, "tracking-service", "startup",
		"Tracking Service started on port "+port, nil)

	addr := ":" + port
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Log(logger.ERROR, "tracking-service", "http-server", "Unable to start server", err)
		os.Exit(1)
	}
}

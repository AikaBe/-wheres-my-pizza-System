package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"restaurant-system/logger"
	"restaurant-system/trackingService/internal/service"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type TrackerHandler struct {
	trackerService *service.TrackerService
}

func NewTrackerHandler(trackerService *service.TrackerService) *TrackerHandler {
	return &TrackerHandler{trackerService: trackerService}
}

func (t *TrackerHandler) GetTrackerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Method not allowed: "+r.Method, nil)
		return
	}
	logger.Log(logger.INFO, "tracking-service", "http-request", "Method: "+r.Method, nil)
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 3 && parts[2] == "status" {
		t.GetOrders(w, r, parts[1])
	} else if len(parts) == 3 && parts[2] == "history" {
		t.GetOrdersHistory(w, r, parts[1])
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid path"})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Invalid path: "+path, nil)
		return

	}
}

func (t *TrackerHandler) GetOrders(w http.ResponseWriter, r *http.Request, number string) {
	w.Header().Set("Content-Type", "application/json")

	result, err := t.trackerService.GetOrderByIdService(number)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Invalid order number ", err)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.Log(logger.ERROR, "tracking-service", "http-response", "Failed to encode response", err)
	}
}

func (t *TrackerHandler) GetOrdersHistory(w http.ResponseWriter, r *http.Request, number string) {
	w.Header().Set("Content-Type", "application/json")

	result, err := t.trackerService.GetHistoryService(number)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Invalid order number ", err)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.Log(logger.ERROR, "tracking-service", "http-response", "Failed to encode response", err)
	}
}

func (t *TrackerHandler) GetWorkers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Method not allowed: "+r.Method, nil)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if parts[1] != "status" && len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid path"})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Invalid path: "+path, nil)
		return
	}

	result, err := t.trackerService.GetWorkers()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		logger.Log(logger.ERROR, "tracking-service", "http-request", "Invalid worker number ", err)
		return
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.Log(logger.ERROR, "tracking-service", "http-response", "Failed to encode response", err)
	}
}

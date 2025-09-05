package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"wheres-my-pizza/orderService/internal/domain"
	"wheres-my-pizza/orderService/internal/service"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type OrderHandler struct {
	orderService *service.OrderService
	semaphore    chan struct{}
}

func NewOrderHandler(orderService *service.OrderService, maxConcurrent int) *OrderHandler {
	return &OrderHandler{orderService: orderService, semaphore: make(chan struct{}, maxConcurrent)}
}

func (handler *OrderHandler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case handler.semaphore <- struct{}{}:
		defer func() { <-handler.semaphore }()
	default:
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Too many concurrent requests"})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		slog.Error("Method not allowed", slog.String("method", r.Method))
		return
	}
	var req domain.CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request: " + err.Error()})
		slog.Error("Invalid request", "error", err)
		return
	}
	orderResponse, err := handler.orderService.CreateOrderService(req)
	if err != nil {
		slog.Error("CreateOrderService error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(orderResponse); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

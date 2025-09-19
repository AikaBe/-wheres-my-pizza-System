package handler

import (
	"encoding/json"
	"net/http"

	"restaurant-system/logger"
	"restaurant-system/orderService/internal/domain"
	"restaurant-system/orderService/internal/service"
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
		logger.Log(logger.ERROR, "order-service", "http-request", "Too many concurrent requests", nil)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		logger.Log(logger.ERROR, "order-service", "http-request", "Method not allowed: "+r.Method, nil)
		return
	}
	var req domain.CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request: " + err.Error()})
		logger.Log(logger.ERROR, "order-service", "decode-body", "Invalid request", err)
		return
	}
	orderResponse, err := handler.orderService.CreateOrderService(req)
	if err != nil {
		logger.Log(logger.ERROR, "order-service", "create-order", "CreateOrderService error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(orderResponse); err != nil {
		logger.Log(logger.ERROR, "order-service", "encode-response", "Failed to encode response", err)
	}
}

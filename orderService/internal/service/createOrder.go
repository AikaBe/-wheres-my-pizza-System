package service

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"restaurant-system/orderService/internal/domain"
)

var (
	mu       sync.Mutex
	lastDate string
	counter  int
)

type OrderService struct {
	orderRepo OrderRepo
	rabbit    RabbitRepo
}

func NewOrderService(orderRepo OrderRepo, rabbit RabbitRepo) *OrderService {
	return &OrderService{orderRepo: orderRepo, rabbit: rabbit}
}

func (str *OrderService) CreateOrderService(req domain.CreateOrderRequest) (domain.OrderResponce, error) {
	err := ValidateOrder(req)
	if err != nil {
		return domain.OrderResponce{}, err
	}

	total := totalAmount(req)
	priority := priority(total)
	number := generateNum()

	responce, err := str.orderRepo.CreateOrderAdapter(req, total, priority, number)
	if err != nil {
		return domain.OrderResponce{}, errors.New("cannot enter data to the DB: " + err.Error())
	}

	if err := str.rabbit.PublishOrder(req, number, priority, total); err != nil {
		return domain.OrderResponce{}, errors.New("cannot publish order: " + err.Error())
	}
	return responce, nil
}

func ValidateOrder(req domain.CreateOrderRequest) error {
	if len(req.CustomerName) < 1 || len(req.CustomerName) > 100 {
		return errors.New("customer name must be between 1 and 100")
	}
	if len(req.Items) < 1 || len(req.Items) > 20 {
		return errors.New("items must be between 1 and 20")
	}
	for _, item := range req.Items {
		if len(item.Name) < 1 || len(item.Name) > 50 {
			return errors.New("item name must be between 1 and 50")
		}
		if item.Quantity < 1 || item.Quantity > 10 {
			return errors.New("item quantity must be between 1 and 10")
		}
		if item.Price < 0.01 || item.Price > 999.99 {
			return errors.New("item price must be between 0.01 and 999.99")
		}
	}

	switch req.Type {
	case "dine_in":
		if req.TableNumber == 0 {
			return errors.New("tableNumber must be set for 'dine_in'")
		} else if req.TableNumber < 0 || req.TableNumber > 100 {
			return errors.New("tableNumber must be between 0 and 100")
		}
	case "takeout":
	case "delivery":
		if req.DeliveryAddress == "" {
			return errors.New("deliveryAddress must be set for 'delivery'")
		} else if len(req.DeliveryAddress) < 10 || len(req.DeliveryAddress) > 100 {
			return errors.New("deliveryAddress must be between 10 and 100")
		}
	default:
		return errors.New("type must be set for 'dine_in', 'takeout' or 'delivery'")
	}

	return nil
}

func generateNum() string {
	mu.Lock()
	defer mu.Unlock()
	today := time.Now().UTC().Format("20060102")
	if lastDate != today {
		lastDate = today
		counter = 0
	}
	counter++
	date := "ORD" + "_" + today + "_" + strconv.Itoa(counter)
	return date
}

func totalAmount(req domain.CreateOrderRequest) float64 {
	total := 0.0
	tempAmount := 0.0
	for _, item := range req.Items {
		tempAmount = float64(item.Quantity) * item.Price
		total += tempAmount
	}
	return total
}

func priority(total float64) int {
	priority := 1
	if total < 50.0 {
		priority = 1
	} else if total > 50.0 && total < 100.0 {
		priority = 5
	} else if total > 100.0 {
		priority = 10
	}
	return priority
}

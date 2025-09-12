package postgre

import (
	"context"
	"restaurant-system/orderService/internal/domain"
)

func (db *OrderService) CreateOrderAdapter(req domain.CreateOrderRequest, total float64, priority int, number string) (domain.OrderResponce, error) {
	ctx := context.Background()

	tx, err := db.conn.Begin(ctx)
	if err != nil {
		return domain.OrderResponce{}, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	_, err = db.conn.Exec(ctx, `insert into orders (created_at,updated_at,number,customer_name,type,table_number,delivery_address,total_amount,priority,processed_by,completed_at) values (Now(),Now(),$1,$2,$3,$4,$5,$6,$7,$8,NULL)`,
		number, req.CustomerName, req.Type, req.TableNumber, req.DeliveryAddress, total, priority, "Aika")
	if err != nil {
		return domain.OrderResponce{}, err
	}

	orderId, err := db.GetOrderId(number)
	if err != nil {
		return domain.OrderResponce{}, err
	}
	for _, item := range req.Items {
		_, err := db.conn.Exec(ctx, `insert into order_items (created_at,order_id,name,quantity,price) 
values (Now(),$1,$2,$3,$4)`, orderId, item.Name, item.Quantity, item.Price)
		if err != nil {
			return domain.OrderResponce{}, err
		}
	}

	_, err = db.conn.Exec(ctx, `insert into order_status_log (created_at,order_id,status,notes)
values (Now(),$1,$2,$3)`, orderId, "received", "created new order")
	if err != nil {
		return domain.OrderResponce{}, err
	}
	row := db.conn.QueryRow(ctx, `select number,status,total_amount from orders order by created_at desc limit 1`)

	var response domain.OrderResponce
	err = row.Scan(&response.Number, &response.Status, &response.TotalAmount)
	if err != nil {
		return domain.OrderResponce{}, err
	}
	return response, nil
}

func (db *OrderService) GetOrderId(number string) (int, error) {
	var id int
	err := db.conn.QueryRow(
		context.Background(),
		`select id from orders where number = $1`,
		number,
	).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

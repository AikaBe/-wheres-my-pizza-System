package postgre

import "github.com/jackc/pgx/v5"

type OrderService struct {
	conn *pgx.Conn
}

func NewOrderRepo(conn *pgx.Conn) *OrderService {
	return &OrderService{conn: conn}
}

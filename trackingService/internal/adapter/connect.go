package adapter

import "github.com/jackc/pgx/v5"

type TrackingService struct {
	conn *pgx.Conn
}

func NewTrackerRepo(conn *pgx.Conn) *TrackingService {
	return &TrackingService{conn: conn}
}

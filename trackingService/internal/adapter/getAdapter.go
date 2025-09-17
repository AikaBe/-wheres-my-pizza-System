package adapter

import (
	"context"
	"restaurant-system/trackingService/internal/domain"
)

func (db *TrackingService) GetOrderById(id string) (domain.TrackingService, error) {
	row := db.conn.QueryRow(context.Background(),
		`select number,status,updated_at,completed_at,processed_by from orders where number = $1`, id)

	var response domain.TrackingService
	err := row.Scan(&response.OrderNumber, &response.CurrentStatus, &response.UpdatedAt, &response.EstimatedCompletion, &response.ProcessedBy)
	if err != nil {
		return domain.TrackingService{}, err
	}
	return response, nil
}

func (db *TrackingService) GetHistory(id string) ([]domain.History, error) {
	number, err := db.GetOrderId(id)
	if err != nil {
		return []domain.History{}, err
	}
	rows, err := db.conn.Query(context.Background(),
		`select status, created_at,changed_by from order_status_log where order_id = $1`, number)
	if err != nil {
		return []domain.History{}, err
	}
	defer rows.Close()

	var history []domain.History
	for rows.Next() {
		var historyRow domain.History
		err := rows.Scan(&historyRow.Status, &historyRow.Timestamp, &historyRow.ChangedBy)
		if err != nil {
			return []domain.History{}, err
		}
		history = append(history, historyRow)
	}
	return history, nil
}

func (db *TrackingService) GetWorkers() ([]domain.Workers, error) {
	rows, err := db.conn.Query(context.Background(), `select name,status,orders_processed,last_seen from workers`)
	if err != nil {
		return []domain.Workers{}, err
	}
	defer rows.Close()
	var workers []domain.Workers
	for rows.Next() {
		var worker domain.Workers
		err := rows.Scan(&worker.WorkerName, &worker.Status, &worker.OrdersProcessed, &worker.LastSeen)
		if err != nil {
			return []domain.Workers{}, err
		}
		workers = append(workers, worker)
	}
	return workers, nil
}

func (db *TrackingService) GetOrderId(number string) (int, error) {
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

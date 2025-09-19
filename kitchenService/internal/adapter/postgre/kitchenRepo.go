package postgre

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type KitchenRepo struct {
	db *pgx.Conn
}

func NewKitchenRepo(db *pgx.Conn) *KitchenRepo {
	return &KitchenRepo{db: db}
}

func (r *KitchenRepo) RegisterWorker(ctx context.Context, workerName, workerType string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var existingStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM workers WHERE name = $1
	`, workerName).Scan(&existingStatus)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	if err == nil && existingStatus == "online" {
		return fmt.Errorf("worker %s is already online", workerName)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workers (name, "type", status, last_seen, orders_processed)
		VALUES ($1, $2, 'online', NOW(), 0)
		ON CONFLICT (name) 
		DO UPDATE SET status = 'online',
		              last_seen = NOW(),
		              "type" = EXCLUDED."type"
	`, workerName, workerType)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *KitchenRepo) UpdateWorkerHeartbeat(ctx context.Context, workerName string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE workers SET last_seen = NOW() WHERE name = $1
	`, workerName)
	return err
}

func (r *KitchenRepo) SetWorkerOffline(ctx context.Context, workerName string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE workers SET status = 'offline', last_seen = NOW() WHERE name = $1
	`, workerName)
	return err
}

func (r *KitchenRepo) UpdateOrderStatus(ctx context.Context, orderNumber, status, processedBy string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE orders SET status = $1, processed_by = $2, updated_at = NOW()
		WHERE number = $3
	`, status, processedBy, orderNumber)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_log (order_id, status, changed_by, changed_at, notes)
		SELECT id, $1, $2, NOW(), $3
		FROM orders WHERE number = $4
	`, status, processedBy, "Status changed by "+processedBy, orderNumber)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *KitchenRepo) CompleteOrder(ctx context.Context, orderNumber, processedBy string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE orders 
		SET status = 'ready', processed_by = $1, completed_at = NOW(), updated_at = NOW()
		WHERE number = $2
	`, processedBy, orderNumber)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_log (order_id, status, changed_by, changed_at, notes)
		SELECT id, 'ready', $1, NOW(), 'Order completed'
		FROM orders WHERE number = $2
	`, processedBy, orderNumber)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE workers SET orders_processed = orders_processed + 1 WHERE name = $1
	`, processedBy)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *KitchenRepo) GetOrderCurrentStatus(ctx context.Context, orderNumber string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status FROM orders WHERE number = $1
	`, orderNumber).Scan(&status)
	return status, err
}

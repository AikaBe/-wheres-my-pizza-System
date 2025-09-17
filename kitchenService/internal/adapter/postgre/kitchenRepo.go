package postgre

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type KitchenRepo struct {
	pool *pgxpool.Pool
}

func NewKitchenRepo(pool *pgxpool.Pool) *KitchenRepo {
	return &KitchenRepo{pool: pool}
}

// 🔹 Обновление статуса заказа и логирование
func (r *KitchenRepo) UpdateOrderStatus(ctx context.Context, orderID string, status, worker string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Обновляем заказ
	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = $1, 
			processed_by = $2,
			completed_at = CASE WHEN $1 = 'ready' THEN now() ELSE completed_at END
		WHERE id = $3
	`, status, worker, orderID)
	if err != nil {
		slog.Error("Failed to update order", "error", err, "order_id", orderID)
		return err
	}

	// Логируем изменение
	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_log (order_id, status, changed_at)
		VALUES ($1, $2, now())
	`, orderID, status)
	if err != nil {
		slog.Error("Failed to insert status log", "error", err, "order_id", orderID)
		return err
	}

	// Если заказ завершён — увеличиваем счётчик обработанных заказов
	if status == "ready" {
		_, err = tx.Exec(ctx, `
			UPDATE workers
			SET orders_processed = orders_processed + 1
			WHERE name = $1
		`, worker)
		if err != nil {
			slog.Error("Failed to increment orders processed", "error", err, "worker", worker)
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		return err
	}

	return nil
}

// 🔹 Получение текущего статуса заказа
func (r *KitchenRepo) GetOrderStatus(ctx context.Context, orderID string) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		slog.Error("Failed to get order status", "error", err, "order_id", orderID)
		return "", err
	}
	return status, nil
}

// 🔹 Начало готовки заказа
func (r *KitchenRepo) StartCookingOrder(ctx context.Context, orderID string, worker string) error {
	return r.UpdateOrderStatus(ctx, orderID, "cooking", worker)
}

// 🔹 Завершение заказа
func (r *KitchenRepo) CompleteOrder(ctx context.Context, orderID string, worker string) error {
	return r.UpdateOrderStatus(ctx, orderID, "ready", worker)
}

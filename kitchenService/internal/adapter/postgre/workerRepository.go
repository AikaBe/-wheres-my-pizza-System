package postgre

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerInfo struct {
	ID              int
	CreatedAt       pgx.NullTime
	Name            string
	Type            string
	Status          string
	LastSeen        pgx.NullTime
	OrdersProcessed int
}

type WorkerRepository struct {
	pool *pgxpool.Pool
}

func NewWorkerRepository(pool *pgxpool.Pool) *WorkerRepository {
	return &WorkerRepository{pool: pool}
}

// 🔹 Регистрация или обновление воркера
func (r *WorkerRepository) RegisterWorker(ctx context.Context, name, workerType string) error {
	query := `
		INSERT INTO workers (name, type, status, last_seen)
		VALUES ($1, $2, 'online', NOW())
		ON CONFLICT (name) 
		DO UPDATE SET
			type = EXCLUDED.type,
			status = CASE 
				WHEN workers.status = 'offline' THEN 'online'
				ELSE workers.status
			END,
			last_seen = EXCLUDED.last_seen
		RETURNING status
	`

	var currentStatus string
	err := r.pool.QueryRow(ctx, query, name, workerType).Scan(&currentStatus)
	if err != nil {
		slog.Error("RegisterWorker error", "error", err)
		return err
	}

	if currentStatus == "online" {
		return ErrWorkerAlreadyActive
	}

	slog.Info("Worker registered/updated successfully", "name", name, "type", workerType)
	return nil
}

// 🔹 Проверка, активен ли воркер с данным именем
func (r *WorkerRepository) IsWorkerActive(ctx context.Context, name string) (bool, error) {
	query := `
		SELECT status = 'online' 
		FROM workers 
		WHERE name = $1
	`

	var isActive bool
	err := r.pool.QueryRow(ctx, query, name).Scan(&isActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		slog.Error("IsWorkerActive error", "error", err)
		return false, err
	}

	return isActive, nil
}

// 🔹 Отправка heartbeat (обновление last_seen)
func (r *WorkerRepository) SendHeartbeat(ctx context.Context, name string) error {
	query := `
		UPDATE workers 
		SET last_seen = NOW(), status = 'online'
		WHERE name = $1
	`

	result, err := r.pool.Exec(ctx, query, name)
	if err != nil {
		slog.Error("SendHeartbeat error", "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Warn("Worker not found for heartbeat", "name", name)
		return ErrWorkerNotFound
	}

	return nil
}

// 🔹 Обновление статуса воркера
func (r *WorkerRepository) UpdateWorkerStatus(ctx context.Context, name, status string) error {
	query := `
		UPDATE workers 
		SET status = $1, last_seen = NOW()
		WHERE name = $2
	`

	result, err := r.pool.Exec(ctx, query, status, name)
	if err != nil {
		slog.Error("UpdateWorkerStatus error", "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Warn("Worker not found for status update", "name", name)
		return ErrWorkerNotFound
	}

	slog.Info("Worker status updated", "name", name, "status", status)
	return nil
}

// 🔹 Получение информации о воркере
func (r *WorkerRepository) GetWorker(ctx context.Context, name string) (*WorkerInfo, error) {
	query := `
		SELECT id, created_at, name, type, status, last_seen, orders_processed
		FROM workers 
		WHERE name = $1
	`

	var worker WorkerInfo
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&worker.ID,
		&worker.CreatedAt,
		&worker.Name,
		&worker.Type,
		&worker.Status,
		&worker.LastSeen,
		&worker.OrdersProcessed,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWorkerNotFound
		}
		slog.Error("GetWorker error", "error", err)
		return nil, err
	}

	return &worker, nil
}

// 🔹 Увеличение счетчика обработанных заказов
func (r *WorkerRepository) IncrementOrdersProcessed(ctx context.Context, name string) error {
	query := `
		UPDATE workers 
		SET orders_processed = orders_processed + 1, last_seen = NOW()
		WHERE name = $1
	`

	result, err := r.pool.Exec(ctx, query, name)
	if err != nil {
		slog.Error("IncrementOrdersProcessed error", "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Warn("Worker not found for incrementing orders", "name", name)
		return ErrWorkerNotFound
	}

	return nil
}

// 🔹 Получение всех активных воркеров
func (r *WorkerRepository) GetActiveWorkers(ctx context.Context) ([]WorkerInfo, error) {
	query := `
		SELECT id, created_at, name, type, status, last_seen, orders_processed
		FROM workers 
		WHERE status = 'online'
		ORDER BY last_seen DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		slog.Error("GetActiveWorkers error", "error", err)
		return nil, err
	}
	defer rows.Close()

	var workers []WorkerInfo
	for rows.Next() {
		var worker WorkerInfo
		err := rows.Scan(
			&worker.ID,
			&worker.CreatedAt,
			&worker.Name,
			&worker.Type,
			&worker.Status,
			&worker.LastSeen,
			&worker.OrdersProcessed,
		)
		if err != nil {
			slog.Error("Scan worker error", "error", err)
			continue
		}
		workers = append(workers, worker)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Rows error", "error", err)
		return nil, err
	}

	return workers, nil
}

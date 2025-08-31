package storage

import (
	"context" // контекст
	"errors"  // подключение возможности создавать свои ошибки
	"time"    // подключение времени

	"github.com/jackc/pgx/v5/pgxpool" // пул соединений для БД
)

var ErrNotFound = errors.New("order not found") // создаем свою новую ошибку

// определение структуры для репозитория заказов
type OrderRepo struct {
	pool *pgxpool.Pool
}

// определение конструктора
func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{pool: pool}
}

// возвращает JSON по UID
func (r *OrderRepo) GetByUID(ctx context.Context, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var raw []byte
	// защита от SQL-инъекции с помощью плейсходера + чтение сырых байтов JSON
	err := r.pool.QueryRow(ctx, `SELECT data FROM orders WHERE order_uid=$1`, uid).Scan(&raw)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return raw, nil
}

func (r *OrderRepo) Upsert(ctx context.Context, uid string, rawJSON []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO orders (order_uid, data)
		VALUES ($1, $2::jsonb)
		ON CONFLICT (order_uid) DO UPDATE
		SET data = EXCLUDED.data, updated_at = NOW()
		`, uid, rawJSON)
	return err
}

// функция для прогрева кеша
func (r *OrderRepo) ListRecent(ctx context.Context, limit int) (map[string][]byte, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT order_uid, data
		FROM orders.orders
		ORDER BY updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]byte, limit)
	for rows.Next() {
		var uid string
		var raw []byte
		if err := rows.Scan(&uid, &raw); err != nil {
			return nil, err
		}
		out[uid] = raw
	}

	return out, rows.Err()
}

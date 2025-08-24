package storage
 
import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool" // подключение пула соединений
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Настройка конфига пула соединений для БД
	cfg.MinConns = 1
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg) // Открытие пула с настроенным конфигом
	if err != nil {
		return nil, err
	}

	// пингуем созданный пул, чтобы проверить что он работает
	// ранний health-check
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second) // дерайвим контекст с таймаутом от родительского
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
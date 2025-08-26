package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOrdersRepo_UpsertAndGetByUID(t *testing.T) {
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		t.Skip("PG_TEST_DSN is empty; set it in .env")
	}

	pool, err := NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	repo := NewOrderRepo(pool)

	uid :="test-" + time.Now().Format("20060102150405")
	raw := []byte(`{"order_uid": "` + uid + `", "track_number": "TST123"}`)

	if err := repo.Upsert(context.Background(), uid, raw); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByUID(context.Background(), uid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(raw) {
		t.Fatalf("mismatch:\n got=%s\nwant=%s", got, raw)
	}

	_, _ = pool.Exec(context.Background(), `DELETE FROM orders.orders WHERE order_uid=$1`, uid)
}
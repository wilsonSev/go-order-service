package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wilsonSev/go-order-service/internal/api"
	"github.com/wilsonSev/go-order-service/internal/storage"
)

func main() {
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN is empty")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := storage.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db pool: %v", err) // Пишет лог и моментально завершает программу с ненулевым кодом
	}
	defer pool.Close()

	repo := storage.NewOrderRepo(pool)
	srv := &http.Server{
		Addr:         ":8081",
		Handler:      api.NewServer(repo),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// запускаем ListenAndServe фоном в отдельной горутине
	go func() {
		log.Printf("HTTP listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// ждем, пока пользователь не завершит выполнение
	<-ctx.Done()
	log.Println("shutting down...")
	
	// graceful shutdown
	// даем серверу 5 секунд, чтобы завершить активные процессы
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}
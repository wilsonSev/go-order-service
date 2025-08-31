package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wilsonSev/go-order-service/internal/api"
	"github.com/wilsonSev/go-order-service/internal/cache"
	"github.com/wilsonSev/go-order-service/internal/kafka"
	"github.com/wilsonSev/go-order-service/internal/storage"
)

func main() {
	log.Println("main started")
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

	cached := cache.NewCachedOrders(repo, repo, 10*time.Minute, 3*time.Second)
	defer cached.Stop()

	warmCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	recent, err := repo.ListRecent(warmCtx, 1000)
	if err != nil {
		log.Printf("warmup: skip (%v)", err)
	} else {
		cached.BulkSet(recent)
		log.Printf("warmup: cached %d orders", len(recent))
	}

	// Инициализация Kafka consumer
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		kafkaTopic = "orders"
	}

	kafkaGroupID := os.Getenv("KAFKA_GROUP_ID")
	if kafkaGroupID == "" {
		kafkaGroupID = "order-service-group"
	}

	consumer := kafka.New(kafka.Config{
		Brokers:     strings.Split(kafkaBrokers, ","),
		Topic:       kafkaTopic,
		GroupID:     kafkaGroupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     500 * time.Millisecond,
		StartOffset: 0, // kafka.FirstOffset
	}, cached)
	defer consumer.Close()

	// Запуск Kafka consumer в отдельной горутине
	go func() {
		log.Printf("Kafka consumer started for topic: %s, group: %s", kafkaTopic, kafkaGroupID)
		if err := consumer.Run(ctx); err != nil {
			log.Printf("kafka consumer error: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      api.NewServer(cached),
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

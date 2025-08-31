package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/wilsonSev/go-order-service/internal/codec"
)

// Upserter — то, во что мы будем писать (DB-репо или кеш-обёртка с write-through).
type Upserter interface {
	Upsert(ctx context.Context, uid string, rawJSON []byte) error
}

type Consumer struct {
	reader   *kafka.Reader
	upserter Upserter
}

type Config struct {
	Brokers     []string
	Topic       string
	GroupID     string
	MinBytes    int
	MaxBytes    int
	MaxWait     time.Duration
	StartOffset int64
}

func New(cfg Config, up Upserter) *Consumer {
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 10e3
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10e6
	}
	if cfg.MaxWait == 0 {
		cfg.MaxWait = 500 * time.Millisecond
	}
	if cfg.StartOffset == 0 {
		cfg.StartOffset = kafka.FirstOffset
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.GroupID,
		Topic:       cfg.Topic,
		MinBytes:    cfg.MinBytes,
		MaxBytes:    cfg.MaxBytes,
		MaxWait:     cfg.MaxWait,
		StartOffset: cfg.StartOffset,
	})
	return &Consumer{reader: r, upserter: up}
}

func (c *Consumer) Close() error { return c.reader.Close() }

func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("kafka: fetch error: %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		raw := m.Value

		// 1) Парсинг и валидация
		o, err := codec.ParseAndValidate(raw)
		if err != nil {
			log.Printf("kafka: invalid message (partition=%d offset=%d): %v", m.Partition, m.Offset, err)
			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("kafka: commit failed after invalid: %v", err)
			}
			continue
		}

		// 2) Запись в БД (идемпотентный upsert по order_uid)
		if err := c.upserter.Upsert(ctx, o.OrderUID, raw); err != nil {
			log.Printf("kafka: upsert failed (uid=%s, partition=%d offset=%d): %v",
				o.OrderUID, m.Partition, m.Offset, err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("kafka: commit failed: %v", err)
		}
	}
}

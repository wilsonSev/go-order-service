package cache

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Для Read-Through кеша
type OrderGetter interface {
	GetByUID(ctx context.Context, uid string) ([]byte, error)
}

// Для Write-Through кеша
type OrderUpserter interface {
	Upsert(ctx context.Context, uid string, rawJSON []byte) error
}

var ErrNotFound = errors.New("order not found")

type entry struct {
	data    []byte
	expires time.Time
	neg     bool // negative cache
}

type CachedOrders struct {
	base      OrderGetter
	upserter  OrderUpserter
	ttl       time.Duration      // Time to live
	negTTL    time.Duration      // Time
	mu        sync.RWMutex       // Read-Write Mutex позволяет многим читателям получать доступ к актуальным данным
	items     map[string]entry   // Сам кеш
	sf        singleflight.Group // Защита от stampede при miss
	cleanQuit chan struct{}      // остановка джанитора(уборщика неактальных данных)
}

func NewCachedOrders(base OrderGetter, upserter OrderUpserter, ttl, negTTL time.Duration) *CachedOrders {
	c := &CachedOrders{
		base:      base,
		upserter:  upserter,
		ttl:       ttl,
		negTTL:    negTTL,
		items:     make(map[string]entry),
		cleanQuit: make(chan struct{}),
		// mu и sf инициализируются по-умолчанию нулевыми значениями
	}

	go c.janitor(1 * time.Minute)
	return c
}

func (c *CachedOrders) janitor(period time.Duration) {
	t := time.NewTicker(period) // канал, в который периодически приходит текущее время
	defer t.Stop()
	for {
		select {
		case <-t.C:
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.items {
				if now.After(v.expires) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.cleanQuit:
			return
		}
	}
}

func (c *CachedOrders) Stop() { close(c.cleanQuit) }

func (c *CachedOrders) GetByUID(ctx context.Context, uid string) ([]byte, error) {
	now := time.Now()
	c.mu.RLock()
	if e, ok := c.items[uid]; ok && now.Before(e.expires) {
		c.mu.RUnlock()
		if e.neg {
			return nil, ErrNotFound
		}
		log.Println("cache HIT", uid)
		// Возвращаем копию, т.к. срез передается по ссылке
		cp := make([]byte, len(e.data))
		copy(cp, e.data)
		return cp, nil
	}
	c.mu.RUnlock()

	v, err, _ := c.sf.Do(uid, func() (any, error) {
		c.mu.RLock()
		if e, ok := c.items[uid]; ok && now.Before(e.expires) {
			c.mu.RUnlock()
			if e.neg {
				return nil, ErrNotFound
			}
			cp := make([]byte, len(e.data))
			copy(cp, e.data)
			return cp, nil
		}
		c.mu.RUnlock()

		data, err := c.base.GetByUID(ctx, uid)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				if c.negTTL > 0 {
					c.mu.Lock()
					c.items[uid] = entry{expires: time.Now().Add(c.negTTL), neg: true}
					c.mu.Unlock()
				}
				return nil, ErrNotFound
			}
		}
		log.Println("cache MISS", uid)
		cp := make([]byte, len(data))
		copy(cp, data)
		c.mu.Lock()
		c.items[uid] = entry{data: cp, expires: time.Now().Add(c.ttl)}
		c.mu.Unlock()
		return cp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (c *CachedOrders) Upsert(ctx context.Context, uid string, rawJSON []byte) error {
	if c.upserter == nil {
		c.mu.Lock()
		delete(c.items, uid)
		c.mu.Unlock()
		return nil
	}
	if err := c.upserter.Upsert(ctx, uid, rawJSON); err != nil {
		return err
	}

	cp := make([]byte, len(rawJSON))
	copy(cp, rawJSON)
	c.mu.Lock()
	c.items[uid] = entry{data: cp, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return nil
}

func (c *CachedOrders) Invalidate(uid string) {
	c.mu.Lock()
	delete(c.items, uid)
	c.mu.Unlock()
}

// выгружаем несколько записей в одной функции
func (c *CachedOrders) BulkSet(m map[string][]byte) {
	now := time.Now()
	c.mu.Lock()
	for k, v := range m {
		cp := make([]byte, len(v))
		copy(cp, v)
		c.items[k] = entry{data: cp, expires: now.Add(c.ttl)}
	}
	c.mu.Unlock()
}

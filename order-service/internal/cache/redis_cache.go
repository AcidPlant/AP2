package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}

type RedisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisOrderCache(client *redis.Client, ttl time.Duration) OrderCache {
	return &RedisOrderCache{client: client, ttl: ttl}
}

func cacheKey(id string) string {
	return fmt.Sprintf("order:%s", id)
}

func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	val, err := c.client.Get(ctx, cacheKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		// Cache miss – not an error, just absent
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis GET: %w", err)
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}
	return &order, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	if err := c.client.Set(ctx, cacheKey(order.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis SET: %w", err)
	}
	log.Printf("[Cache] SET  order:%s  (ttl=%s)", order.ID, c.ttl)
	return nil
}

func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, cacheKey(id)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis DEL: %w", err)
	}
	log.Printf("[Cache] DEL  order:%s  (invalidated)", id)
	return nil
}

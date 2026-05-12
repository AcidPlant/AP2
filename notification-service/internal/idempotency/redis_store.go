package idempotency

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	statusProcessing = "processing"
	statusDone       = "done"
	defaultTTL       = 24 * time.Hour
)

type RedisStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &RedisStore{rdb: rdb, ttl: ttl}
}

func idemKey(eventID string) string {
	return fmt.Sprintf("notif:idem:%s", eventID)
}

func (s *RedisStore) TryAcquire(ctx context.Context, eventID string) (acquired bool, err error) {
	key := idemKey(eventID)
	ok, err := s.rdb.SetNX(ctx, key, statusProcessing, s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency SetNX: %w", err)
	}
	if !ok {
		val, err := s.rdb.Get(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return false, fmt.Errorf("idempotency GET: %w", err)
		}
		if val == statusDone {
			log.Printf("[Idempotency] DUPLICATE (done)  event=%s – skipping", eventID)
			return false, nil
		}
		log.Printf("[Idempotency] STALE processing state for event=%s – re-acquiring", eventID)
		return true, nil
	}
	log.Printf("[Idempotency] ACQUIRED event=%s", eventID)
	return true, nil
}

func (s *RedisStore) MarkDone(ctx context.Context, eventID string) error {
	key := idemKey(eventID)
	if err := s.rdb.Set(ctx, key, statusDone, s.ttl).Err(); err != nil {
		return fmt.Errorf("idempotency MarkDone: %w", err)
	}
	log.Printf("[Idempotency] DONE event=%s", eventID)
	return nil
}

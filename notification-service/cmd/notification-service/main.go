package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"notification-service/internal/consumer"
	"notification-service/internal/idempotency"
	"notification-service/internal/provider"
	"notification-service/internal/worker"

	"github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("notification-service: cannot connect to Redis at %s: %v", redisAddr, err)
	}
	log.Printf("Redis connected at %s", redisAddr)

	idemTTLHours, _ := strconv.Atoi(getEnv("IDEMPOTENCY_TTL_HOURS", "24"))
	idemStore := idempotency.New(rdb, time.Duration(idemTTLHours)*time.Hour)

	mode := strings.ToUpper(getEnv("PROVIDER_MODE", "SIMULATED"))
	var notifProvider provider.NotificationProvider
	switch mode {
	case "REAL":
		notifProvider = provider.NewSMTPProvider()
		log.Println("Provider: REAL (SMTP)")
	default:
		latencyMs, _ := strconv.Atoi(getEnv("MOCK_LATENCY_MS", "200"))
		failureRate, _ := strconv.ParseFloat(getEnv("MOCK_FAILURE_RATE", "0.3"), 64)
		notifProvider = provider.NewMockProvider(latencyMs, failureRate)
		log.Printf("Provider: SIMULATED (latency=%dms, failure_rate=%.0f%%)", latencyMs, failureRate*100)
	}

	maxRetries, _ := strconv.Atoi(getEnv("RETRY_MAX_ATTEMPTS", "4"))
	baseDelayMs, _ := strconv.Atoi(getEnv("RETRY_BASE_DELAY_MS", "2000"))
	maxDelayMs, _ := strconv.Atoi(getEnv("RETRY_MAX_DELAY_MS", "30000"))

	workerCfg := worker.Config{
		MaxRetries: maxRetries,
		BaseDelay:  time.Duration(baseDelayMs) * time.Millisecond,
		MaxDelay:   time.Duration(maxDelayMs) * time.Millisecond,
	}
	notifWorker := worker.New(idemStore, notifProvider, workerCfg)
	log.Printf("Worker config: max_retries=%d base_delay=%dms max_delay=%dms",
		maxRetries, baseDelayMs, maxDelayMs)

	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	c, err := consumer.New(rabbitURL, notifWorker)
	if err != nil {
		log.Fatalf("notification-service: consumer init: %v", err)
	}
	defer func(c *consumer.RabbitMQConsumer) {
		err := c.Close()
		if err != nil {

		}
	}(c)

	runCtx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		log.Printf("notification-service: received signal %v – shutting down…", sig)
		cancel()
	}()

	log.Println("notification-service: started")
	if err := c.Consume(runCtx); err != nil {
		log.Fatalf("notification-service: consume error: %v", err)
	}
	log.Println("notification-service: stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

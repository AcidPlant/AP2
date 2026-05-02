package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/consumer"
	"notification-service/internal/handler"
	"notification-service/internal/idempotency"
)

func main() {
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	idem := idempotency.New()
	notifHandler := handler.New(idem)

	c, err := consumer.New(rabbitURL, notifHandler)
	if err != nil {
		log.Fatalf("notification-service: consumer init: %v", err)
	}
	defer func(c *consumer.RabbitMQConsumer) {
		err := c.Close()
		if err != nil {

		}
	}(c)

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		log.Printf("notification-service: received signal %v – shutting down…", sig)
		cancel()
	}()

	log.Println("notification-service: started")
	if err := c.Consume(ctx); err != nil {
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

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"order-service/internal/client"
	"order-service/internal/repository"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dsn := getEnv("DATABASE_URL", "postgres://postgres:123@localhost:5432/orders_db?sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	paymentURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")

	// Custom HTTP client with 2-second timeout as required.
	httpClient := &http.Client{Timeout: 2 * time.Second}

	// Manual Dependency Injection (Composition Root).
	paymentClient := client.NewPaymentClient(paymentURL, httpClient)
	orderRepo := repository.NewPostgresRepo(db)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)
	handler := transporthttp.NewHandler(orderUC)

	r := gin.Default()
	handler.RegisterRoutes(r)

	port := getEnv("PORT", "8080")
	log.Printf("order-service listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

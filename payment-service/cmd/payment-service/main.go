package main

import (
	"database/sql"
	"log"
	"os"

	"payment-service/internal/repository"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dsn := getEnv("DATABASE_URL", "postgres://postgres:123@localhost:5432/payments_db?sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Manual Dependency Injection (Composition Root).
	paymentRepo := repository.NewPostgresRepo(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)
	handler := transporthttp.NewHandler(paymentUC)

	r := gin.Default()
	handler.RegisterRoutes(r)

	port := getEnv("PORT", "8081")
	log.Printf("payment-service listening on :%s", port)
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

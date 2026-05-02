package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-service/internal/broker"
	"payment-service/internal/middleware"
	"payment-service/internal/repository"
	transportgrpc "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	paymentv1 "github.com/AcidPlant/generated-code/payment/v1"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dsn := getEnv("DATABASE_URL", "postgres://postgres:123@localhost:5432/payments_db?sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	pub, err := broker.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("rabbitmq publisher: %v", err)
	}
	defer func(pub *broker.RabbitMQPublisher) {
		err := pub.Close()
		if err != nil {

		}
	}(pub)

	paymentRepo := repository.NewPostgresRepo(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo, pub)

	grpcPort := getEnv("GRPC_PORT", "9091")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UnaryLoggingInterceptor),
	)
	paymentv1.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentGRPCServer(paymentUC))
	reflection.Register(grpcServer)

	go func() {
		log.Printf("payment-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	handler := transporthttp.NewHandler(paymentUC)
	r := gin.Default()
	handler.RegisterRoutes(r)

	httpPort := getEnv("PORT", "8081")
	srv := &http.Server{Addr: ":" + httpPort, Handler: r}

	go func() {
		log.Printf("payment-service HTTP listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("payment-service: shutting down…")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	log.Println("payment-service: stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

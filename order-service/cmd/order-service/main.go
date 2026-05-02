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

	"order-service/internal/broker"
	"order-service/internal/client"
	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	orderv1 "github.com/AcidPlant/generated-code/order/v1"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	dsn := getEnv("DATABASE_URL", "postgres://postgres:123@localhost:5432/orders_db?sslmode=disable")
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

	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:9091")
	paymentClient, err := client.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("payment grpc client: %v", err)
	}

	orderRepo := repository.NewPostgresRepo(db)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)

	orderBroker := broker.NewOrderBroker()
	go orderBroker.ListenAndForward(dsn)

	grpcPort := getEnv("GRPC_PORT", "9090")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}
	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, transportgrpc.NewOrderStreamServer(orderRepo, orderBroker))

	go func() {
		log.Printf("order-service gRPC streaming on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	handler := transporthttp.NewHandler(orderUC)
	r := gin.Default()
	handler.RegisterRoutes(r)

	httpPort := getEnv("PORT", "8080")
	srv := &http.Server{Addr: ":" + httpPort, Handler: r}

	go func() {
		log.Printf("order-service HTTP listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("order-service: shutting down…")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	log.Println("order-service: stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

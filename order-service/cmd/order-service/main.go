package main

import (
	"database/sql"
	"log"
	"net"
	"os"

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

	port := getEnv("PORT", "8080")
	log.Printf("order-service HTTP listening on :%s", port)
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

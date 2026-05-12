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
	"strconv"
	"syscall"
	"time"

	"order-service/internal/broker"
	"order-service/internal/cache"
	"order-service/internal/client"
	"order-service/internal/middleware"
	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	orderv1 "github.com/AcidPlant/generated-code/order/v1"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[WARN] redis unavailable at %s: %v – caching disabled", redisAddr, err)
		rdb = nil
	} else {
		log.Printf("Redis connected at %s", redisAddr)
	}

	cacheTTLSec, _ := strconv.Atoi(getEnv("CACHE_TTL_SECONDS", "300"))
	var orderCache cache.OrderCache
	if rdb != nil {
		orderCache = cache.NewRedisOrderCache(rdb, time.Duration(cacheTTLSec)*time.Second)
	}

	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:9091")
	paymentClient, err := client.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("payment grpc client: %v", err)
	}

	orderRepo := repository.NewPostgresRepo(db)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient, orderCache)

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

	r := gin.Default()

	if rdb != nil {
		maxReq, _ := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "10"))
		windowSec, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECONDS", "60"))
		r.Use(middleware.RateLimiter(rdb, middleware.RateLimiterConfig{
			MaxRequests: maxReq,
			Window:      time.Duration(windowSec) * time.Second,
		}))
		log.Printf("Rate limiter enabled: %d req / %ds", maxReq, windowSec)
	}

	handler := transporthttp.NewHandler(orderUC)
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

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	if rdb != nil {
		_ = rdb.Close()
	}

	log.Println("order-service: stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

OVERVIEW

This project has two microservices:

Order Service (port 8080)
Payment Service (port 8081)

The Order Service handles creating and managing orders, and it calls the Payment Service to process payments.

Both services are independent:

each has its own database
they communicate via HTTP
no shared code or tables

ARCHITECTURE

\## Architecture



```

&#x20;                        CLIENT (curl / Postman)

&#x20;                                  |

&#x20;         -----------------------------------------------------

&#x20;         |                                                   |

&#x20;  POST/GET/PATCH /orders                           GET /payments/:order\_id

&#x20;         |                                                   |

&#x20;         v                                                   v



+------------------------------+           +------------------------------+

|        ORDER SERVICE         |  HTTP     |       PAYMENT SERVICE        |

|            :8080             | --------> |            :8081             |

|------------------------------|           |------------------------------|

| Handler (HTTP)               |           | Handler (HTTP)               |

|      |                       |           |      |                       |

|      v                       |           |      v                       |

| OrderUseCase                 |           | PaymentUseCase               |

|      |                       |           |      |                       |

|      v                       |           |      v                       |

| Repository                   |           | Repository                   |

|      |                       |           |      |                       |

|      v                       |           |      v                       |

| orders\_db (PostgreSQL)       |           | payments\_db (PostgreSQL)     |

+------------------------------+           +------------------------------+

```



\*\*Notes:\*\*

\- Order Service calls Payment Service via HTTP  

\- Each service has its own database  

\- No direct DB access between services  Notes:

Order Service calls Payment Service via HTTP
Each service has its own database
No direct DB access between services

HOW THE CODE IS STRUCTURED

Each service follows a Clean Architecture style (simplified):

Handler (transport/http)
Handles HTTP requests and responses
UseCase
Contains business logic
Domain
Just structs (no DB, no HTTP)
Repository
Works with the database
Client (Order Service only)
Calls the Payment Service via HTTP

Important idea:
Everything depends inward → business logic doesn’t depend on frameworks

BOUNDED CONTEXTS

Order Service:

manages orders
uses orders table
calls Payment Service

Payment Service:

manages payments
uses payments table

They do not access each other’s database

BUSINESS RULES

Amount is stored as int64 (in cents)
Amount must be greater than 0 → otherwise 400
If amount > 100000 → payment is declined → order = "Failed"
Only "Pending" orders can be cancelled
if already "Paid" → return 409

WHAT HAPPENS IF PAYMENT SERVICE IS DOWN

There is a 2 second timeout, so requests don’t hang forever
If payment fails:
order is saved as "Failed"
DB write still happens using context.Background()
API returns:
503 Service Unavailable

Why "Failed" and not "Pending"?

Because:

"Pending" means we didn’t try yet
but here we actually tried and it failed

So "Failed" is more accurate and avoids confusion

IDEMPOTENCY (BONUS)

If you send a header like:

Idempotency-Key: order-123

Then:

sending the same request again won’t create duplicates
it returns the same order

This is handled using a UNIQUE column in the database

HOW TO RUN THE PROJECT

Using Docker:

docker-compose up --build

This will:

start both services
start databases
run migrations automatically

Without Docker:

Terminal 1:
cd payment-service
go mod tidy
go run ./cmd/payment-service

Terminal 2:
cd order-service
go mod tidy
go run ./cmd/order-service

API EXAMPLES

Create order (normal case)

POST /orders

Example body:
{
"customer\_id": "cust-1",
"item\_name": "Laptop",
"amount": 50000
}

Result:

201 Created
status = "Paid"

Create order (declined)

If amount > 100000
Result: status = "Failed"

Invalid request

If amount = 0
Result: 400 Bad Request

Get order

GET /orders/<id>

Cancel order

PATCH /orders/<id>/cancel

200 → if Pending
409 → if Paid or Cancelled
404 → if not found

Get payment

GET /payments/<order-id>

Idempotency example

Send same request twice with:

Idempotency-Key: order-abc-123

Result:

same response
no duplicate order

Simulate failure

stop payment-service
send POST /orders

Result:

waits about 2 seconds
returns 503
order is saved as "Failed"

FINAL NOTES

This project demonstrates:
microservice separation
clean architecture basics
failure handling
idempotency
It is simplified but follows real-world concepts

# Assignment 2 — gRPC Migration

## Overview

This project consists of two microservices:

* Order Service (port 8080, REST)
* Payment Service (port 9091, gRPC)

The Order Service is responsible for creating and managing orders.
The Payment Service is responsible for processing payments.

The Order Service communicates with the Payment Service using gRPC.
Each service has its own PostgreSQL database, and there is no direct data sharing between them.

---

## Architecture

### High-Level Flow

```
Client (REST)
     |
     v
Order Service (:8080)
     |
     | gRPC (unary)
     v
Payment Service (:9091)

Streaming:
grpcurl / client ---> Payment Service (:9090)
```

---

### Detailed Architecture

```
                        CLIENT (curl / Postman)
                                  |
                                  v
                          REST API (:8080)
                                  |
                                  v

+------------------------------+
|        ORDER SERVICE         |
|------------------------------|
| HTTP Handler                 |
|      |                       |
|      v                       |
| OrderUseCase                 |
|      |                       |
|      v                       |
| Repository                   |
|      |                       |
|      v                       |
| orders_db (PostgreSQL)       |
|                              |
| gRPC Client ----------------------+
+------------------------------+    |
                                    v
                          +------------------------------+
                          |      PAYMENT SERVICE         |
                          |------------------------------|
                          | gRPC Server (:9091)          |
                          | (unary methods)              |
                          |                              |
                          | PaymentUseCase               |
                          |      |                       |
                          |      v                       |
                          | Repository                  |
                          |      |                       |
                          |      v                       |
                          | payments_db (PostgreSQL)    |
                          |                              |
                          | gRPC Streaming (:9090)       |
                          +------------------------------+
```

---

## Repositories

Proto definitions:
https://github.com/AcidPlant/Proto

Generated gRPC code:
https://github.com/AcidPlant/generated-code

---

## Communication

The Order Service uses gRPC to communicate with the Payment Service.

### Unary Call

Used during order creation to process payment:

```
Order Service → PaymentService.ProcessPayment()
```

### Streaming

Used to receive payment updates:

```
grpcurl → PaymentService.SubscribePayments()
```

---

## Code Structure

Each service follows a simplified clean architecture:

```
Handler → UseCase → Domain → Repository
```

* Handler: processes incoming HTTP or gRPC requests
* UseCase: contains business logic
* Domain: defines core data structures
* Repository: handles database interaction

The Order Service additionally includes a gRPC client for calling the Payment Service.

---

## Migration (HTTP → gRPC)

Before migration:

```
Order Service → HTTP → Payment Service
```

After migration:

```
Order Service → gRPC → Payment Service
```

gRPC is used to improve performance, enforce strict contracts via proto files, and support streaming communication.

---

## Business Rules

* Amount is stored as int64 (in cents)
* Amount must be greater than 0
* If amount > 100000, payment is declined

Order statuses:

* Pending
* Paid
* Failed
* Cancelled

---

## Failure Handling

If the Payment Service is unavailable:

* Request timeout is 2 seconds
* The order is still saved in the database
* Order status is set to "Failed"
* API returns HTTP 503

The status "Failed" is used instead of "Pending" because the system attempted the payment but did not succeed.

---

## Idempotency

Supported using the header:

```
Idempotency-Key: <key>
```

If the same request is repeated with the same key:

* No duplicate order is created
* The same response is returned

This is implemented using a UNIQUE constraint in the database.

---

## Running the Project

### Using Docker

```
docker-compose up --build
```

This starts:

* both services
* PostgreSQL databases
* runs migrations

---

### Manual Run

Terminal 1:

```
cd payment-service
go mod tidy
go run ./cmd/payment-service
```

Terminal 2:

```
cd order-service
go mod tidy
go run ./cmd/order-service
```

---

## API Examples

### Create Order

```
POST /orders
```

Request body:

```
{
  "customer_id": "cust-1",
  "item_name": "Laptop",
  "amount": 50000
}
```

Responses:

* 201 — order created, status "Paid"
* 400 — invalid request
* 503 — payment service unavailable

---

### Get Order

```
GET /orders/{id}
```

---

### Cancel Order

```
PATCH /orders/{id}/cancel
```

* 200 — success
* 409 — already Paid or Cancelled
* 404 — not found

---

### Get Payment

```
GET /payments/{order_id}
```

---

## gRPC Testing

### Unary Request

```
grpcurl -plaintext \
  -d '{"order_id":"123","amount":50000}' \
  localhost:9091 \
  payment.PaymentService/ProcessPayment
```

---

### Streaming Request

```
grpcurl -plaintext \
  -d '{"order_id":"123"}' \
  localhost:9090 \
  payment.PaymentService/SubscribePayments
```

---

## Notes

This project demonstrates:

* microservice separation with independent databases
* clean architecture principles
* gRPC communication (unary and streaming)
* failure handling with timeouts
* idempotency implementation

The implementation is simplified but reflects common real-world patterns.

If the same request is sent again with the same key:

* no duplicate order is created
* the same response is returned

This is implemented using a UNIQUE constraint in the database.

---

## Running the Project

### Using Docker

```
docker-compose up --build
```

This starts:

* both services
* PostgreSQL databases
* migrations

---

### Manual запуск

Terminal 1:

```
cd payment-service
go mod tidy
go run ./cmd/payment-service
```

Terminal 2:

```
cd order-service
go mod tidy
go run ./cmd/order-service
```

---

## API Examples

### Create Order

```
POST /orders
```

Body:

```
{
  "customer_id": "cust-1",
  "item_name": "Laptop",
  "amount": 50000
}
```

Responses:

* 201 → Paid
* 400 → invalid request
* 503 → payment service unavailable

---

### Get Order

```
GET /orders/{id}
```

---

### Cancel Order

```
PATCH /orders/{id}/cancel
```

* 200 → success
* 409 → already Paid or Cancelled
* 404 → not found

---

### Get Payment

```
GET /payments/{order_id}
```

---

## gRPC Testing

### Unary

```
grpcurl -plaintext \
  -d '{"order_id":"123","amount":50000}' \
  localhost:9091 \
  payment.PaymentService/ProcessPayment
```

---

### Streaming

```
grpcurl -plaintext \
  -d '{"order_id":"123"}' \
  localhost:9090 \
  payment.PaymentService/SubscribePayments
```

# AP2 Assignment 3 — Event-Driven Architecture with RabbitMQ

## Architecture Overview

```
┌─────────────────┐   HTTP      ┌──────────────────┐
│   Order Service │ ──────────► │ Payment Service  │
│   (port 8080)   │   gRPC      │   (port 8081)    │
└────────┬────────┘             └────────┬─────────┘
         │                               │  Publish JSON event
         │                               │  (after DB commit)
         ▼                               ▼
  orders-db (PG)        ┌─────────────────────────────┐
                         │         RabbitMQ            │
                         │  Queue: payment.completed   │
                         │  DLX:   payment.dlx         │
                         │  DLQ:   payment.dead        │
                         └────────────┬────────────────┘
                                      │  Consume (manual ACK)
                                      ▼
                         ┌────────────────────────────┐
                         │   Notification Service     │
                         │   - Manual ACK             │
                         │   - Idempotency (sync.Map) │
                         │   - Graceful Shutdown      │
                         └────────────────────────────┘
```

## How to Run

```bash
docker compose up --build
```

RabbitMQ Management UI: http://localhost:15672 (guest / guest)

## Testing

### Create an order (triggers full chain)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":    "cust-001",
    "item_name":      "Laptop",
    "amount":         49999,
    "customer_email": "alice@example.com",
    "idempotency_key":"order-001"
  }'
```

### Watch notification logs
```bash
docker logs -f notification-service
# Expected: [Notification] Sent email to alice@example.com for Order #<id>. Amount: $499.99
```

### Test declined payment (amount > 100000 cents = $1000)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c2","item_name":"Yacht","amount":999999,"customer_email":"bob@example.com"}'
# No notification — declined payments are not published
```

---

## Idempotency Strategy

Each `PaymentEvent` carries a unique `event_id` (UUID). The Notification Service keeps an in-memory `sync.Map` keyed by `event_id`.

```
Handle(body):
  1. Parse JSON → PaymentEvent
  2. MarkSeen(event.EventID) → duplicate? ACK and return immediately
  3. Log the email simulation
  4. Return nil → consumer ACKs the message
```

This prevents duplicate email logs even when RabbitMQ re-delivers a message after a consumer crash.

## Manual ACK Logic

Auto-ack is **disabled**. The consumer:
- Sends `msg.Ack(false)` only after `Handle()` succeeds (returns nil)
- Sends `msg.Nack(false, false)` on error — no requeue, so RabbitMQ routes to DLQ after 3 attempts (`x-delivery-limit: 3`)

This achieves **at-least-once delivery**: a crash before ACK causes re-delivery.

## Dead Letter Queue (Bonus)

| Resource | Type | Role |
|---|---|---|
| `payment.completed` | durable queue | main work queue |
| `payment.dlx` | fanout exchange | dead-letter exchange |
| `payment.dead` | durable queue | receives permanently-failed messages |

To demonstrate: change `handler.Handle()` to return an error, rebuild, and watch `payment.dead` fill up in the RabbitMQ UI.

## Graceful Shutdown

Both services use `os/signal.Notify(quit, SIGINT, SIGTERM)`:
- **Payment Service**: `grpcServer.GracefulStop()` + `http.Server.Shutdown(ctx)` then closes RabbitMQ channel
- **Notification Service**: cancels the consumer `context.Context`, exits the consume loop, `defer c.Close()` releases AMQP resources

## Queue Durability

All queues/exchanges declared `durable: true`. Messages published with `DeliveryMode: amqp.Persistent`. Messages survive broker restarts.

## Project Structure

```
.
├── docker-compose.yml
├── README.md
├── order-service/          (Assignment 2 — unchanged)
├── payment-service/        (Assignment 2 + broker/publisher added)
│   └── internal/broker/
│       ├── publisher.go         ← interface
│       └── rabbitmq_publisher.go
└── notification-service/   (NEW — Assignment 3)
    └── internal/
        ├── consumer/rabbitmq_consumer.go
        ├── handler/notification_handler.go
        └── idempotency/store.go
```


---


AP2 Assignment 4 – Performance Optimization & External Integrations
Overview
---
Architecture
```
┌────────────────────────────────────────────────────────────────────┐
│                        CLIENT (curl / Postman)                     │
└───────────────────────────────┬────────────────────────────────────┘
                                │  HTTP
                                ▼
┌───────────────────────────────────────────────────────┐
│                    ORDER SERVICE :8080                │
│                                                       │
│  RateLimiterMiddleware ──► Handler                    │
│            (Redis)              │                     │
│                                 ▼                     │
│                         OrderUseCase                  │
│                        /           \                  │
│               ┌────────┐       ┌────────┐             │
│               │  Redis │       │Postgres│             │
│               │ Cache  │       │  (DB)  │             │
│               └────────┘       └────────┘             │
│         Cache-aside read:                             │
│           1. Check Redis → HIT → return               │
│           2. MISS → read DB → SET Redis → return      │
│         Cache invalidation:                           │
│           UpdateStatus → DEL Redis key (atomic)       │
│                                                       │
│               │  gRPC Authorize                       │
│               ▼                                       │
└───────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────┐
│                  PAYMENT SERVICE :8081                │
│                                                       │
│  gRPC Server → PaymentUseCase → Postgres              │
│                     │                                 │
│                     │  Publish event (async)          │
│                     ▼                                 │
│              RabbitMQ Publisher                       │
│          (queue: payment.completed)                   │
└───────────────────────────────────────────────────────┘
                │
                │  AMQP message
                ▼
┌───────────────────────────────────────────────────────┐
│              NOTIFICATION SERVICE (worker)            │
│                                                       │
│  RabbitMQConsumer → NotificationWorker                │
│                            │                          │
│            ┌───────────────┼──────────────────┐       │
│            │               │                  │       │
│            ▼               ▼                  ▼       │
│     Redis Idempotency  Exponential       Provider     │
│        Store           Backoff          Interface     │
│     (SetNX / MarkDone) (2s→4s→8s→16s) /           \  │
│                                  MockProvider  SMTP  │
│                                  (simulated)  (real) │
│                                                       │
│  On success → MarkDone in Redis → ACK message        │
│  On failure → retry with backoff → NACK → DLQ        │
└───────────────────────────────────────────────────────┘
```
Redis Usage Map
```
Key pattern               Service       Purpose
─────────────────────     ────────────  ────────────────────────────────────
order:<id>                order-svc     Cache-aside order data (TTL 5 min)
rate_limit:<ip>           order-svc     Request counter for rate limiter
notif:idem:<eventID>      notif-svc     Idempotency state (processing/done)
```
---
Caching Strategy (Order Service)
Pattern: Cache-Aside (Lazy Loading)
The Cache-Aside pattern was chosen because:
Orders are read far more often than written.
We only cache entries that are actually requested, keeping Redis memory lean.
The application (not Redis) owns the invalidation logic, giving us full control.
Read Path (`GET /orders/:id`)
```
Request
  │
  ├─► Redis GET order:<id>
  │       ├─ HIT  → return immediately (no DB call)
  │       └─ MISS ─► Postgres SELECT
  │                       │
  │                       └─► Redis SET order:<id>  TTL=5min
  │                               │
  └───────────────────────────────┘
                  Response
```
Write / Invalidation Path
```
CreateOrder / UpdateStatus / CancelOrder
  │
  ├─► Postgres UPDATE orders SET status = ...
  │
  └─► Redis DEL order:<id>   ← atomic invalidation
```
Why DEL instead of SET?
Deleting the key is safer than overwriting it because:
It avoids stale-read races between two concurrent writers.
The next read will fetch the freshest data from Postgres and re-populate the
cache with the correct TTL.
TTL
Default is 300 seconds (5 minutes), configurable via `CACHE_TTL_SECONDS`.
The TTL acts as a safety net: even if invalidation somehow misses (e.g. a crash
between the DB write and the DEL call), the cache auto-expires within the window.
---
Retry & Backoff Strategy (Notification Service)
Exponential Backoff
```
Attempt 1  → send immediately
Attempt 2  → wait 2 s  then send
Attempt 3  → wait 4 s  then send
Attempt 4  → wait 8 s  then send
...capped at RETRY_MAX_DELAY_MS (default 30 s)
```
Formula: `delay = min(BaseDelay × 2^(attempt-1), MaxDelay)`
Configured via environment variables:
Variable	Default	Meaning
`RETRY_MAX_ATTEMPTS`	`4`	Total attempts (1 initial + 3 retries)
`RETRY_BASE_DELAY_MS`	`2000`	First retry wait
`RETRY_MAX_DELAY_MS`	`30000`	Maximum wait cap
Idempotency
Redis is used as a distributed idempotency store. Each payment event has a unique
`event_id`. The flow:
```
TryAcquire(eventID)
  │
  ├─ SetNX "notif:idem:<id>" = "processing"  TTL=24h
  │
  ├─ OK (new key)    → proceed to send
  └─ Already exists
       ├─ value = "done"       → skip (already delivered)
       └─ value = "processing" → re-acquire (previous crash recovery)

After successful send:
  Set "notif:idem:<id>" = "done"
```
This prevents:
Duplicate emails – same eventID won't be processed twice.
Lost notifications – a "processing" state from a crashed worker is
re-acquired on the next attempt.
---
Adapter Pattern (Provider)
```go
// The interface – only this lives in the worker
type NotificationProvider interface {
    Send(ctx context.Context, n Notification) error
}

// Concrete adapters
MockProvider  – PROVIDER_MODE=SIMULATED (default)
SMTPProvider  – PROVIDER_MODE=REAL
```
Switching providers requires only an environment variable change – zero code
changes to the business logic.
---
Rate Limiter (Bonus)
A middleware in `order-service/internal/middleware/rate_limiter.go` limits each
client (by IP) to N requests per time window using a Redis counter:
```
INCR rate_limit:<ip>     (atomic)
EXPIRE rate_limit:<ip>   <window>

if counter > limit:
    HTTP 429  +  Retry-After header
```
Configure via `.env`:
```
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_WINDOW_SECONDS=60
```
---
Running the Project
Prerequisites
Docker & Docker Compose
Start everything
```bash
docker compose up --build
```
Verify services
Service	URL
Order Service HTTP	http://localhost:8080
Payment Service HTTP	http://localhost:8081
RabbitMQ Management	http://localhost:15672 (guest/guest)
Redis CLI	`docker compose exec redis redis-cli`
Example API calls
```bash
# Create an order (triggers payment + notification)
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-key-001" \
  -d '{"customer_id":"cust-1","item_name":"Widget","amount":4999,"customer_email":"alice@example.com"}'

# Get order – first call hits DB, second call hits cache
ORDER_ID="<id from above>"
curl http://localhost:8080/orders/$ORDER_ID

# Cancel order (also invalidates cache)
curl -X PATCH http://localhost:8080/orders/$ORDER_ID/cancel

# Inspect Redis cache
docker compose exec redis redis-cli KEYS "order:*"
docker compose exec redis redis-cli TTL "order:$ORDER_ID"

# Inspect idempotency keys
docker compose exec redis redis-cli KEYS "notif:idem:*"

# Trigger rate limiter (send > 10 requests in 60 seconds)
for i in $(seq 1 12); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/healthz; done
```
Switch to real SMTP
```bash
# In notification-service/.env:
PROVIDER_MODE=REAL
SMTP_HOST=smtp.mailjet.com
SMTP_PORT=587
SMTP_USER=<mailjet_api_key>
SMTP_PASSWORD=<mailjet_secret>
EMAIL_FROM=noreply@yourdomain.com

docker compose up --build notification-service
```


Final Project Structure
```
AP2_Assignment4/
├── docker-compose.yml
│
├── order-service/
│   ├── cmd/order-service/main.go       ← wires Redis + rate limiter
│   ├── internal/
│   │   ├── cache/redis_cache.go        ← NEW: Cache-Aside implementation
│   │   ├── middleware/rate_limiter.go  ← NEW: Redis rate limiter (BONUS)
│   │   ├── usecase/order_usecase.go    ← UPDATED: uses cache interface
│   │   ├── transport/http/handler.go
│   │   ├── repository/order_repo.go
│   │   ├── domain/order.go
│   │   ├── broker/order_broker.go
│   │   └── client/payment_grpc_client.go
│   ├── migrations/
│   ├── .env
│   ├── Dockerfile
│   └── go.mod
│
├── notification-service/
│   ├── cmd/notification-service/main.go   ← wires provider via PROVIDER_MODE
│   ├── internal/
│   │   ├── provider/
│   │   │   ├── provider.go             ← NEW: NotificationProvider interface
│   │   │   ├── mock_provider.go        ← NEW: simulated adapter
│   │   │   └── smtp_provider.go        ← NEW: real SMTP adapter
│   │   ├── idempotency/redis_store.go  ← NEW: Redis-backed idempotency
│   │   ├── worker/notification_worker.go ← NEW: retry + backoff engine
│   │   └── consumer/rabbitmq_consumer.go ← UPDATED: uses worker
│   ├── .env
│   ├── Dockerfile
│   └── go.mod
│
└── payment-service/                       ← unchanged from Assignment 3
    ├── cmd/payment-service/main.go
    ├── internal/...
    ├── migrations/
    ├── .env
    ├── Dockerfile
    └── go.mod
```
---



## Notes

This project demonstrates:

* basic microservice separation
* clean architecture approach
* gRPC communication (unary and streaming)
* failure handling with timeouts
* idempotency implementation

The implementation is simplified but follows real-world concepts.


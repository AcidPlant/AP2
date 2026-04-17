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

---

## Notes

This project demonstrates:

* basic microservice separation
* clean architecture approach
* gRPC communication (unary and streaming)
* failure handling with timeouts
* idempotency implementation

The implementation is simplified but follows real-world concepts.


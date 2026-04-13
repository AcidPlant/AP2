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

Assignment 2 — gRPC Migration

Repositories
- **Proto Repository (Repo A):** https://github.com/AcidPlant/Proto
- **Generated Code Repository (Repo B):** https://github.com/AcidPlant/generated-code

Architecture
[Client] --REST--> [Order Service :8080] --gRPC--> [Payment Service :9091]
                         
                         |
                   [gRPC Streaming :9090] <-- grpcurl / client script

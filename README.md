# Asset Repayment System

A backend service that processes payment notifications for mobility entrepreneurs repaying productive assets, written in Go. When a payment notification is received, the system validates it, records the payment, and updates the customer's outstanding balance atomically. The service is designed to handle more than **100,000 payment notifications per minute** reliably, with idempotent handling of duplicate notifications.

## Design Decisions & Approach

The core challenge was building a system that could handle **100,000 payment notifications per minute** reliably, without double-counting payments, while keeping the implementation straightforward and dependency-light.

### Approach

The solution is a synchronous HTTP API backed by SQLite. Each incoming notification is validated, written to the database in a single atomic transaction, and the customer's balance is updated in the same transaction — so there is no state where a payment is recorded but the balance is not updated, or vice versa.

Idempotency was a deliberate design priority. Payment providers retry on timeout. Rather than returning `409` on a duplicate reference (which triggers retries), the system returns `200` — the payment was already applied, the outcome is the same. A pre-check handles the common case; the `UNIQUE` constraint on `transaction_reference` handles concurrent races.

### Key decisions

| Decision | Reason |
|----------|--------|
| SQLite with WAL mode | Single-node simplicity with concurrent-read support; measured at ~77,500 req/sec — well above the 100k/min requirement |
| Amounts stored in kobo (int64) | Avoids floating point precision errors in financial calculations |
| Duplicate pre-check + UNIQUE constraint | Defence in depth — pre-check is the fast path; UNIQUE constraint is the safety net for concurrent races |
| Service layer owns the transaction | Keeps repositories composable and testable; transaction boundary is a business concern, not a data concern |
| `net/http` only, no frameworks | Reduces dependency surface; stdlib is sufficient for a single-endpoint API |
| Custom migration runner | Avoids pulling in a migration library for a simple ordered-file pattern |
| Idempotent duplicate handling | Payment providers always retry — returning `200` on a seen reference stops retry loops |

### Tools & Technologies

| Tool | Role |
|------|------|
| Go 1.26 | Language |
| `net/http` | HTTP server (stdlib only) |
| `database/sql` | Database access (stdlib only) |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| SQLite (WAL mode) | Embedded database with concurrent-read support |
| `wrk` | Load testing |

---

## Overview

Each customer is assigned an asset worth **1,000,000 NGN**, repaid over **50 weeks** via bank transfers into virtual accounts. When a payment notification arrives, the system validates it, records the payment, and updates the customer's outstanding balance atomically.

## Project Structure

```
.
├── cmd/server/main.go          # Entry point — wires all layers and starts HTTP server
├── internal/
│   ├── domain/                 # Types, constants, sentinel errors
│   ├── db/                     # SQLite connection and migration runner
│   ├── repository/             # Data access layer (customers, payments)
│   ├── service/                # Business logic and transaction management
│   └── handler/                # HTTP request handling and response mapping
├── migrations/                 # Ordered SQL migration files
└── README.md
```

## Architecture

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  Parse JSON, validate method, map errors → HTTP status
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Service   │  Validate payload, own DB transaction (BEGIN/COMMIT/ROLLBACK)
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│     Repository      │  CustomerRepository + PaymentRepository
│  (takes *sql.Tx)    │  Idempotency check, UNIQUE constraint enforcement
└──────┬──────────────┘
       │
       ▼
┌─────────────┐
│   SQLite    │  WAL mode, busy_timeout=5000ms, foreign keys ON
└─────────────┘
```

## Getting Started

### Prerequisites

- Go 1.26+

### Build

```bash
go build -o assetrepayment ./cmd/server
```

### Run

```bash
./assetrepayment
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:8080` | HTTP listen address |
| `DB_PATH` | `./data.db` | Path to SQLite database file |

```bash
DB_PATH=/var/data/repayment.db PORT=:9090 ./assetrepayment
```

Migrations run automatically on startup.

## API

Base URL: `http://localhost:8080`

---

### POST /payments

Process a payment notification from the payment provider.

**Request headers:**
```
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `customer_id` | string | ✅ | Customer identifier (e.g. `GIGXXXXX`) |
| `payment_status` | string | ✅ | Must be `COMPLETE` to be processed |
| `transaction_amount` | string | ✅ | Amount in **kobo** (integer). 10000 = 100 NGN |
| `transaction_date` | string | ✅ | Format: `YYYY-MM-DD HH:MM:SS` |
| `transaction_reference` | string | ✅ | Unique reference from the payment provider |

---

#### Successful payment

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIGXXXXX",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-07 14:54:16",
    "transaction_reference": "VPAY25110713542114478761522000"
  }'
```

```json
{"status":"ok"}
```

---

#### Non-COMPLETE status (silently ignored)

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIGXXXXX",
    "payment_status": "PENDING",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-07 14:54:16",
    "transaction_reference": "VPAY25110713542114478761522001"
  }'
```

```json
{"status":"ignored"}
```

---

#### Duplicate transaction reference → 200 (idempotent)

Sending the same `transaction_reference` a second time returns success — the payment was already recorded and the balance already updated. This is intentional: payment providers retry on timeout.

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIGXXXXX",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-07 14:54:16",
    "transaction_reference": "VPAY25110713542114478761522000"
  }'
```

```json
{"status":"ok"}
```

---

#### Unknown customer → 404

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIGUNKNOWN",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-07 14:54:16",
    "transaction_reference": "VPAY25110713542114478761522002"
  }'
```

```json
{"error":"customer not found"}
```

---

#### Missing fields → 400

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "GIGXXXXX"}'
```

```json
{"error":"invalid payload: missing required fields"}
```

---

#### Invalid amount → 400

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIGXXXXX",
    "payment_status": "COMPLETE",
    "transaction_amount": "0",
    "transaction_date": "2025-11-07 14:54:16",
    "transaction_reference": "VPAY25110713542114478761522003"
  }'
```

```json
{"error":"invalid transaction amount: amount must be greater than zero"}
```

---

**Response code summary:**

| Status | Condition |
|--------|-----------|
| `200 {"status":"ok"}` | Payment processed successfully, or duplicate (idempotent) |
| `200 {"status":"ignored"}` | `payment_status` is not `COMPLETE` |
| `400` | Missing fields or invalid amount |
| `404` | Customer not found |
| `405` | Wrong HTTP method |
| `500` | Unexpected server error |

## Running Tests

```bash
go test ./...
```

## Throughput

Load tested with [`wrk`](https://github.com/wg/wrk) on a single node:

```
Thread Stats   Avg      Stdev     Max   +/- Stdev
  Latency     2.10ms    7.74ms 221.94ms   97.65%
  Req/Sec    19.55k     3.91k   28.73k    89.54%
4657650 requests in 1.00m, 781.77MB read
Requests/sec: 77,511.42
```

**~77,500 req/sec (~4.6M/minute) on a single node.** Average latency of 2.1ms with 97.65% of requests within one standard deviation.

SQLite in WAL mode supports concurrent readers and a single serialised writer. With `_txlock=immediate` and `_busy_timeout=5000`, writers queue rather than fail. Each write transaction is small (~3 SQL statements). `db.SetMaxOpenConns(1)` prevents `SQLITE_BUSY` errors under concurrent goroutines, and `UNIQUE(transaction_reference)` provides a DB-level guard against duplicate payments under race conditions.



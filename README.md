# Asset Repayment System

A backend service that processes payment notifications for mobility entrepreneurs repaying productive assets.

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

#### Duplicate transaction reference → 409

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
{"error":"duplicate payment: transaction reference already processed"}
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
| `200 {"status":"ok"}` | Payment processed successfully |
| `200 {"status":"ignored"}` | `payment_status` is not `COMPLETE` |
| `400` | Missing fields or invalid amount |
| `404` | Customer not found |
| `409` | Duplicate `transaction_reference` |
| `405` | Wrong HTTP method |
| `500` | Unexpected server error |

## Running Tests

```bash
go test ./...
```

## Scaling to 100,000 Payments per Minute

100,000 payments/minute = ~1,667 requests/second.

### Why SQLite works here (with caveats)

SQLite in WAL mode supports concurrent readers and a single writer. With `_txlock=immediate` and `_busy_timeout=5000`, writers queue rather than fail instantly. Each write transaction is small (~3 SQL statements), completing in under 1ms on local disk.

**Single-node estimate:**
- 1ms per write → ~1,000 writes/second per single writer
- That gives ~60,000/minute on one machine — close, but not enough headroom

### Path to 100k/minute

**Option 1 — Queue-fronted workers (recommended for production):**

```
Payment Provider
      │
      ▼
  HTTP Ingress (stateless, many instances)
      │  write to queue
      ▼
  Kafka / Redis Streams
      │  consume in parallel
      ▼
  Worker Pool (N goroutines per node)
      │
      ▼
  SQLite (or Postgres for multi-writer scale)
```

The HTTP layer returns `202 Accepted` immediately. Workers process asynchronously. This decouples ingestion throughput from DB write speed.

**Option 2 — Batch writes:**

Buffer payments in memory for 10–50ms and flush in a single transaction. Reduces lock contention dramatically. Suitable if slight processing delay is acceptable.

**Option 3 — Migrate to PostgreSQL:**

Replace `modernc.org/sqlite` with `lib/pq` or `pgx`. The repository and service layers are unchanged — only `db.Open` and the DSN change. Postgres supports many concurrent writers and horizontal read replicas.

### Concurrency safety in this implementation

- `db.SetMaxOpenConns(1)` — serialises all writes through one connection, preventing SQLite `SQLITE_BUSY` under concurrent goroutines
- `UNIQUE(transaction_reference)` — DB-level guard against duplicates even under race conditions
- `_txlock=immediate` — writer acquires the write lock at `BEGIN`, not at first write, preventing deadlocks
- Service layer owns `BEGIN/COMMIT/ROLLBACK` — no partial updates possible

## Design Decisions

| Decision | Reason |
|----------|--------|
| Amounts stored in kobo (int64) | Avoids floating point precision errors in financial calculations |
| Duplicate check before insert + UNIQUE constraint | Defence in depth — pre-check is fast; UNIQUE is the safety net under race conditions |
| Service owns the transaction | Repository methods stay composable and testable; transaction boundary is a business concern |
| No external frameworks | `net/http` + `database/sql` are sufficient and reduce operational surface area |
| Migration runner is custom | Avoids external dependency for a simple ordered-file pattern |

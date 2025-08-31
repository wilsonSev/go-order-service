# Go Order Service
https://github.com/user-attachments/assets/663d765c-e143-4b65-b53a-dc52ea80d3b0

## A microservice which:
- consumes orders from Kafka (JSON messages)
- validates the payload
- upserts into PostgreSQL (jsonb + indexes)
- caches in memory (read-through / write-through)
- serves order data by order_uid over HTTP API
- provides a simple web UI for searching orders

## Quick start

### Clone the repo

```bash
git clone https://github.com/wilsonSev/go-order-service.git
cd go-order-service
```

### Prepare environment (you can create your own environment variables)

```bash
cp .env.example .env
```
### Start infrastructure (Postgres, Redpanda, Kafka UI)

```bash
docker compose up -d
```

### Export env vars

```bash
export $(cat .env | xargs)
```

### Run DB migrations

```bash
migrate -database "$PG_DSN" -path migrations up
```

### Start the service

```bash
go run ./cmd/app
```

### You can use Kafka UI for creating orders using link http://localhost:8080

### Use web interface to get orders by id using link http://localhost:8081

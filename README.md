# Crypto Service

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Tracing-000000?logo=opentelemetry&logoColor=white)

A backend service written in Go for fetching cryptocurrency prices from CoinGecko, storing rate history in PostgreSQL, and exposing current and aggregated rates through a REST API.

## Features

- Fetches cryptocurrency prices from the CoinGecko API
- Stores cryptocurrency rate history in PostgreSQL
- Returns current rates for requested cryptocurrencies
- Calculates aggregated rates: `min`, `max`, and `avg`
- Periodically actualizes prices with a background cron job
- REST API built with `chi`
- PostgreSQL access through `pgx`
- Structured logging with Go `slog`
- Distributed tracing with OpenTelemetry and Jaeger
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Docker Compose environment for the service, PostgreSQL, and Jaeger
- Database migrations
- Unit, database integration, and HTTP-level tests
- `golangci-lint` configuration and Taskfile commands for development

## Architecture

```text
                         +----------------+
                         |   CoinGecko    |
                         +-------+--------+
                                 |
                                 v
+--------+     HTTP      +-------+--------+      +------------+
| Client | ------------> |   Go Service   | ---> | PostgreSQL |
+--------+                +-------+--------+      +------------+
                                 |
                                 | OpenTelemetry
                                 v
                         +-------+--------+
                         |     Jaeger     |
                         +----------------+

                  Cron job -> price actualization
```

The project separates transport, business logic, domain entities, and infrastructure adapters. Dependencies are connected in the application composition root.

## Tech Stack

- **Language:** Go 1.25
- **HTTP:** `go-chi/chi`
- **Database:** PostgreSQL 16, `pgx/v5`
- **External API:** CoinGecko
- **Scheduling:** `gocron`
- **Tracing:** OpenTelemetry + Jaeger
- **Configuration:** Koanf + YAML + environment variables
- **Testing:** Go `testing`, Testify
- **Infrastructure:** Docker Compose
- **Tooling:** Taskfile, golangci-lint, golang-migrate

## API

Base URL when running locally:

```text
http://localhost:9001/crypto/v1
```

### Get current rates

```http
POST /crypto/v1/rates
Content-Type: application/json
```

Request:

```json
{
  "titles": ["btc", "eth"]
}
```

Example response:

```json
{
  "coins": [
    {
      "title": "btc",
      "cost": 63097.42,
      "actual_at": "2026-06-09T12:30:00Z"
    }
  ]
}
```

### Get aggregated rates

```http
POST /crypto/v1/rates/aggregated?aggregate=avg
Content-Type: application/json
```

Supported aggregation types:

```text
avg | min | max
```

Request:

```json
{
  "titles": ["btc", "eth"]
}
```

## Running Locally

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- [Task](https://taskfile.dev/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- CoinGecko API token

### 1. Clone the repository

```bash
git clone https://github.com/chernrom/crypto_service.git
cd crypto_service
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Set your CoinGecko token in `.env`:

```env
COIN_GECKO_TOKEN=your_token_here
```

### 3. Start the project

```bash
task up
```

The API will be available at:

```text
http://localhost:9001
```

Jaeger UI:

```text
http://localhost:16686
```

### 4. Stop the project

```bash
task down
```

## Example Request

```bash
curl -X POST http://localhost:9001/crypto/v1/rates \
  -H "Content-Type: application/json" \
  -d '{"titles":["btc","eth"]}'
```

Aggregated rates:

```bash
curl -X POST "http://localhost:9001/crypto/v1/rates/aggregated?aggregate=avg" \
  -H "Content-Type: application/json" \
  -d '{"titles":["btc","eth"]}'
```

## Testing

Run Go tests:

```bash
go test ./...
```

Run database integration tests:

```bash
task test_l1
```

Run HTTP-level tests against the complete service stack:

```bash
task test_l2
```

Run the linter:

```bash
task lint
```

## Project Structure

```text
.
├── cmd/app/                     # Application entry point
├── internal/
│   ├── adapters/                # PostgreSQL, CoinGecko, configuration
│   ├── cases/                   # Business logic / use cases
│   ├── entities/                # Domain entities and errors
│   └── port/                    # Service contracts and HTTP transport
├── pkg/
│   ├── application/             # Dependency wiring and application lifecycle
│   └── dto/                     # API request/response DTOs
├── migrations/                  # PostgreSQL migrations
├── test/l2/                     # HTTP-level tests
├── toolkit/tracing/             # OpenTelemetry helpers
├── config/                      # Service configuration
├── deploy/                      # Dockerfile and deployment files
├── compose.yaml                 # Local infrastructure
└── Taskfile.yml                 # Development commands
```

## Reliability and Observability

The service uses request timeouts, structured logging, OpenTelemetry spans, and graceful shutdown. Background rate actualization runs on a configurable schedule and uses context timeouts to prevent hanging operations.

## License

This repository is an educational and portfolio project.

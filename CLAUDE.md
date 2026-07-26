# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 0. Context

This project listens to updates from marketplaces (DBA.dk — Danish classifieds) and has a user interacting with a Telegram bot. An AWS Bedrock LLM (Amazon Nova 2 Lite) analyzes fetched listings for deals worth buying. The frontend's sole purpose was for testing and may be decommissioned.

## 1. Commands

### Repository-wide Commands

```bash
make build                                # Build all Lambda artifacts (Quarkus Native + Go ARM64)
make test                                 # Run unit tests across all backend services
make clean                                # Clean all service build artifacts
```

### Service-specific Commands

- **DBA Ingestor (`services/dba`)**:
  ```bash
  make dev-dba                            # Start Quarkus dev mode (or `quarkus dev` inside services/dba)
  quarkus test                            # Run unit tests inside services/dba
  quarkus build --native                  # Build native Linux binary inside services/dba
  ```

- **Deal Analyzer (`services/analyzer`)**:
  ```bash
  cd services/analyzer && go test ./...   # Run unit tests
  cd services/analyzer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap ./cmd/analyzer  # Build Lambda binary
  ```

- **Telegram Notifier (`services/telegram-notifier`)**:
  ```bash
  cd services/telegram-notifier && go test ./...  # Run unit tests
  cd services/telegram-notifier && GOOS=linux GOARCH=arm64 go build -o build/bootstrap ./cmd/notifier  # Build Lambda binary
  ```

- **Frontend (`front`)**:
  ```bash
  cd front && npm run dev                 # Start Vite dev server (port 3000)
  cd front && npm run build               # Build production bundle
  npx shadcn@latest add <component>       # Add shadcn component to frontend
  ```

- **Prerequisites**: Backend requires Java 25 & Go 1.26+. Frontend uses `npm`.

## 2. Architecture

```
front/                     # Frontend (not actively deployed) — TanStack Router + React + Vite + Tailwind v4
services/                  # Backends
  services/dba/            # DBA updates poller — Quarkus Lambda (Quarkus Native, Amazon Linux 2023)
  services/analyzer/       # Deal Analyzer — Go Lambda (ARM64, Amazon Linux 2023) + Bedrock + DynamoDB
  services/telegram-notifier/# Telegram Notifier — Go Lambda (ARM64, Amazon Linux 2023) + SSM Parameter Store
terraform/                 # AWS infra (Lambda, S3, IAM, SQS, DynamoDB, SSM) — deploys to eu-central-1
```

### Backend flow (Lambda pipeline)

1. `dba-ingestor` (Java/Quarkus) — Triggered by EventBridge schedule; calls DBA.dk search API in parallel (`DbaRestClient`) and publishes JSON Array batch events to `dba-events` SQS queue.
2. `dba-analyzer` (Go) — Triggered by SQS batch; checks DynamoDB (`dba-analyzed-events`) via `BatchGetItem`, analyzes new listings using Amazon Bedrock Nova 2 Lite, and publishes good deals to `deal-notifications` SQS queue.
3. `dba-telegram-notifier` (Go) — Triggered by `deal-notifications` SQS queue; retrieves bot token from AWS SSM Parameter Store and sends HTML alert via Telegram Bot API.

### Frontend

Single-page app hitting `/dba/search` via axios. Validates with Zod schemas in `front/src/interactions/schemas/dba.ts`. API base URL set via `VITE_DBA_API_URL` (defaults to `http://localhost:8080`).

## 3. Data Model

### Pipeline Events

#### `DbaEvent` (`dba-events` SQS payload)
Published by `dba-ingestor` as a singleton JSON Object or JSON Array batch `[ {...} ]`:
```json
{
  "eventId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "id": "12345678",
  "heading": "iPhone 15 Pro 128GB",
  "price": { "amount": 4500, "currencyCode": "DKK" },
  "location": "København N",
  "canonicalUrl": "https://www.dba.dk/iphone-15-pro/id-12345678/",
  "imageUrls": ["https://img.dba.dk/1.jpg"],
  "brand": "Apple",
  "timestamp": 1719000000,
  "extras": [{ "id": "1", "label": "Condition", "values": ["Som ny"] }],
  "query": "iphone 15 pro"
}
```

#### `DealNotification` (`deal-notifications` SQS payload)
Published by `dba-analyzer` when LLM verdict is `GOOD_DEAL`:
```json
{
  "eventId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "heading": "iPhone 15 Pro 128GB",
  "price": { "amount": 4500, "currencyCode": "DKK" },
  "canonicalUrl": "https://www.dba.dk/iphone-15-pro/id-12345678/",
  "location": "København N",
  "verdict": "GOOD_DEAL",
  "reason": "Price is >35% below market average for iPhone 15 Pro in Denmark.",
  "query": "iphone 15 pro"
}
```

#### `LLMVerdict` (Internal Bedrock Response)
Structured response returned from Amazon Bedrock (`eu.amazon.nova-2-lite-v1:0`):
```json
{
  "verdict": "GOOD_DEAL", // or "NOT_A_DEAL"
  "reason": "brief explanation"
}
```

### Storage Schemas

#### DynamoDB (`dba-analyzed-events`)
Used for deduplicating processed events:
- **Partition Key**: `id` (String - offer ID)
- **Attributes**: `analyzedAt` (String RFC3339), `verdict` (String), `expiresAt` (Number, Unix Epoch TTL)
- **TTL**: Auto-deletes records after configured expiration (default: 5 days via `DEDUP_TTL_DAYS`).

## 4. Rules

- Never run `terraform apply`, `terraform destroy`, or any Terraform command that modifies infrastructure. Terraform deployments must be performed manually by the developer.
- After modifying Terraform files, check style and validity locally before committing: run `terraform fmt` and `terraform validate`, and optionally run `tflint` or other linters. Do NOT run `terraform apply` here; planning and apply must be done in a controlled deployment environment.
- Use `quarkus` commands rather than `mvn` or `mvnw`
- All code must be documented following industry standards (godoc for Go, Javadoc for Java, etc.). A feature is not considered implemented until its tests pass.
- Never run git commands. Committing, pushing, branching, and all other git operations must be performed manually by the developer.
- In `services/dba`, if a POJO has a lot of attributes, use a class with a Lombok `@Builder` annotation instead of a Java `record`.
- Coding style: prefer `<` to `>` when it comes to adding code.

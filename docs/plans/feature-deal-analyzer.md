# Feature: Deal Analyzer Pipeline

| Field | Value |
|-------|-------|
| Status | Draft |
| Author | — |
| Created | 2026-07-25 |
| Last Updated | 2026-07-25 |

## 1. Problem Statement

The current DBA Lambda is a monolith that fetches listings, evaluates deals with an LLM, and partially integrates Telegram — all in a single synchronous invocation. This makes it impossible to:
- Process listings independently (a slow Bedrock call blocks everything)
- Avoid re-analyzing the same listing on every run
- Notify users reliably (Telegram is half-implemented)
- Scale or retry each concern independently

## 2. Proposed Solution

Split into 3 event-driven services connected by SQS queues:

```
EventBridge (schedule)
      |
      v
[DBA Ingestor] ──SQS:dba-events──> [Analyzer] ──SQS:deal-notifications──> [Telegram Notifier]
   (Java/Quarkus)                      (Go)                                     (Go)
                                        |
                                   DynamoDB (dedup)
                                   Bedrock (LLM)
```

## 3. Cost Strategy

This is a hobby project. All infrastructure choices prioritize minimal cost:

| Resource | Pricing Approach |
|----------|-----------------|
| Lambda (all 3) | ARM64 (`provided.al2023` for Go), minimum memory (128MB for Go, 512MB for Java due to JVM) |
| SQS | Standard queues (not FIFO) — free tier covers 1M requests/month |
| DynamoDB | On-Demand (PAY_PER_REQUEST) — no cost at zero traffic, pennies at low volume |
| EventBridge | Free for scheduled rules |
| Bedrock (Nova Lite) | Pay-per-token only — no provisioned throughput |

**Estimated monthly cost at low volume (~100 invocations/day): < $1 USD**

---

## 4. User Stories

### Epic 1: DBA Ingestor — Refactor to Pure Data Fetcher

---

#### US-1.1: Remove non-ingestion concerns from DBA service

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | None |

**As** a developer,
**I want** the DBA Lambda to only fetch listings and publish them to a queue,
**So that** it has a single responsibility and downstream services can evolve independently.

**Acceptance Criteria:**
- [ ] `LlmService.java`, `BedrockRuntimeClientService.java`, `TelegramBotService.java` are deleted
- [ ] `DbaHandler` no longer references LLM or Telegram
- [ ] Bedrock SDK and `java-telegram-bot-api` dependencies removed from `pom.xml`
- [ ] Project compiles and existing DBA fetch logic still works

**Test Plan:**
- Unit: `DbaHandler` invoked with a mock `DbaService` → returns results without errors
- Integration: `./mvnw test` passes with no compilation errors after removals

---

#### US-1.2: Publish fetched listings to SQS

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-4.1 (queue must exist) |

**As** the Analyzer service,
**I want** each fetched listing published as an individual SQS message,
**So that** I can process them asynchronously with independent retry semantics.

**Acceptance Criteria:**
- [ ] `SqsPublisher` CDI bean wraps `SqsClient.sendMessageBatch()`
- [ ] Each `DbaDoc` is serialized to JSON matching the `dba-events` schema (see Section 6)
- [ ] `eventId` is derived from `adId` (string representation)
- [ ] Originating search query included in message body
- [ ] Queue URL injected via `DBA_EVENTS_QUEUE_URL` env var
- [ ] Quarkus Amazon SQS extension added to `pom.xml`

**Test Plan:**
- Unit: Mock `SqsClient`, assert `sendMessageBatch` called with correct message bodies
- Integration: Quarkus DevServices with LocalStack; verify messages land in local SQS queue
- Manual: Invoke Lambda locally with `sam local invoke`, check queue via AWS CLI

**Terraform:**
- Add `sqs:SendMessage` and `sqs:GetQueueUrl` to Ingestor IAM role for `dba-events` queue ARN
- Add `DBA_EVENTS_QUEUE_URL` to Lambda environment variables

---

#### US-1.3: Schedule DBA Ingestor via EventBridge

| Field | Value |
|-------|-------|
| Priority | Medium |
| Estimate | S |
| Dependencies | US-1.2 |

**As** a user,
**I want** the Ingestor to run automatically on a schedule,
**So that** I get deals discovered without manual intervention.

**Acceptance Criteria:**
- [ ] EventBridge rule triggers Lambda every 15 minutes (configurable)
- [ ] Event payload contains the search queries list
- [ ] Lambda permission allows EventBridge to invoke it

**Test Plan:**
- Terraform: `terraform plan` shows rule, target, and permission resources
- E2E: After deploy, wait for one scheduled invocation; confirm messages in queue via CloudWatch logs

**Terraform:**
- `aws_cloudwatch_event_rule` with `rate(15 minutes)`
- `aws_cloudwatch_event_target` pointing to Ingestor Lambda with JSON input
- `aws_lambda_permission` for `events.amazonaws.com`

---

### Epic 2: Analyzer Service — Go Lambda

---

#### US-2.1: Scaffold Go Lambda project

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | None |

**As** a developer,
**I want** a working Go Lambda project at `services/analyzer/`,
**So that** I have the foundation for the deal analysis service.

**Acceptance Criteria:**
- [ ] `services/analyzer/` with `go.mod`, `main.go`, `Makefile`
- [ ] Uses `github.com/aws/aws-lambda-go` for Lambda runtime
- [ ] Handler accepts `events.SQSEvent`
- [ ] `Makefile` produces a zip for Lambda deployment (ARM64 binary)
- [ ] Compiles and runs locally with a test SQS event JSON

**Test Plan:**
- `make build` produces `bootstrap` binary
- `go test ./...` passes (basic handler test with mock event)

**Terraform:**
- `aws_lambda_function` resource: runtime `provided.al2023`, architecture `arm64`, memory 128MB, timeout 30s
- S3 object for the deployment zip
- Event source mapping from `dba-events` queue (batch size 5)

---

#### US-2.2: Deduplicate already-processed events

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-2.1, US-4.2 |

**As** the Analyzer service,
**I want** to skip listings that have already been analyzed,
**So that** users don't receive duplicate notifications and LLM costs are minimized.

**Acceptance Criteria:**
- [ ] Before calling Bedrock, check DynamoDB for the event's `eventId`
- [ ] If exists → skip processing, delete message from batch
- [ ] DynamoDB `PutItem` uses `attribute_not_exists(eventId)` condition to handle races
- [ ] TTL set to 7 days from analysis time
- [ ] DynamoDB table name injected via `DEDUP_TABLE_NAME` env var

**Test Plan:**
- Unit: Mock DynamoDB client; test skip-on-existing, write-on-new, conditional-write-failure paths
- Integration: LocalStack DynamoDB; write an event, then re-process — confirm it's skipped
- Edge case: Concurrent writes for same `eventId` — one succeeds, other gracefully skips

**Terraform:**
- `aws_dynamodb_table` with hash key `eventId` (S), TTL on `expiresAt`, PAY_PER_REQUEST
- Analyzer IAM role: `dynamodb:GetItem`, `dynamodb:PutItem` on table ARN

---

#### US-2.3: Analyze listings with Bedrock LLM

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-2.1 |

**As** a user,
**I want** an LLM to evaluate whether a listing is a good deal,
**So that** I only get notified about genuinely underpriced items.

**Acceptance Criteria:**
- [ ] Calls Bedrock Converse API with model `eu.amazon.nova-2-lite-v1:0`
- [ ] Prompt includes: item heading, price, location, brand, extras/condition, search query context
- [ ] System prompt instructs model to respond in JSON: `{"verdict": "GOOD_DEAL"|"NOT_A_DEAL", "reason": "..."}`
- [ ] Response is parsed; malformed responses default to `NOT_A_DEAL`
- [ ] Bedrock client uses default credential chain (IAM role in prod, profile in dev)

**Test Plan:**
- Unit: Mock Bedrock client; verify prompt construction and JSON response parsing
- Unit: Malformed LLM response → graceful fallback to `NOT_A_DEAL`
- Integration: Call real Bedrock in dev with sample listing data; verify structured response

**Terraform:**
- Analyzer IAM role: `bedrock:InvokeModel` on Nova 2 Lite inference profile ARN

---

#### US-2.4: Publish good deals to notification queue

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | US-2.3, US-4.1 |

**As** the Telegram Notifier,
**I want** to receive confirmed good deals on the `deal-notifications` queue,
**So that** I can notify the user without re-evaluating listings.

**Acceptance Criteria:**
- [ ] Only listings with `GOOD_DEAL` verdict are published
- [ ] Message body matches `deal-notifications` schema (see Section 6)
- [ ] Queue URL injected via `DEAL_NOTIFICATIONS_QUEUE_URL` env var

**Test Plan:**
- Unit: Verdict `GOOD_DEAL` → message published; `NOT_A_DEAL` → no publish
- Integration: LocalStack SQS; run handler with test events; verify messages arrive in notification queue

**Terraform:**
- Analyzer IAM role: `sqs:SendMessage` on `deal-notifications` queue ARN
- Analyzer Lambda env var: `DEAL_NOTIFICATIONS_QUEUE_URL`

---

#### US-2.5: Record analyzed events in DynamoDB

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | US-2.2, US-2.3 |

**As** the Analyzer service,
**I want** to persist the analysis result after processing (regardless of verdict),
**So that** future invocations skip already-seen listings.

**Acceptance Criteria:**
- [ ] After analysis, write to DynamoDB: `eventId`, `analyzedAt` (ISO 8601), `verdict`, `expiresAt` (epoch seconds, now + 7 days)
- [ ] Write happens after LLM call but before publishing to notification queue (prevents notify-without-dedup)
- [ ] If conditional write fails (race), skip notification publish (another invocation handled it)

**Test Plan:**
- Unit: Verify DynamoDB item structure and TTL calculation
- Integration: Process an event → confirm DynamoDB item exists with correct fields

---

#### US-2.6: Report partial batch failures

| Field | Value |
|-------|-------|
| Priority | Medium |
| Estimate | S |
| Dependencies | US-2.1 |

**As** an operator,
**I want** only failed messages to retry (not the entire batch),
**So that** successfully processed messages aren't re-delivered.

**Acceptance Criteria:**
- [ ] Handler returns `SQSBatchResponse` with `BatchItemFailures`
- [ ] Only message IDs that errored are included in the failure list
- [ ] Successfully processed messages (including dedup-skipped) are not retried

**Test Plan:**
- Unit: Batch of 3 messages; 1 fails Bedrock call → only that message ID in failures
- Integration: Send 5 messages to LocalStack queue, poison one → confirm 4 succeed, 1 retries

**Terraform:**
- Event source mapping: `function_response_types = ["ReportBatchItemFailures"]`

---

### Epic 3: Telegram Notifier — Go Lambda

---

#### US-3.1: Scaffold Telegram Notifier Go Lambda

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | None |

**As** a developer,
**I want** a working Go Lambda project at `services/telegram-notifier/`,
**So that** I have the foundation for the notification service.

**Acceptance Criteria:**
- [ ] `services/telegram-notifier/` with `go.mod`, `main.go`, `Makefile`
- [ ] Handler accepts `events.SQSEvent` from `deal-notifications` queue
- [ ] `Makefile` produces ARM64 zip for Lambda deployment
- [ ] Compiles and tests pass

**Test Plan:**
- `make build` succeeds
- `go test ./...` passes with basic handler test

**Terraform:**
- `aws_lambda_function`: runtime `provided.al2023`, architecture `arm64`, memory 128MB, timeout 10s
- Event source mapping from `deal-notifications` queue (batch size 1)

---

#### US-3.2: Send deal notification to Telegram

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-3.1 |

**As** a user,
**I want** to receive a Telegram message when a good deal is found,
**So that** I can act on it quickly before someone else buys it.

**Acceptance Criteria:**
- [ ] Bot sends message to configured chat ID via Telegram Bot API (`sendMessage` endpoint)
- [ ] Bot token from `TELEGRAM_BOT_TOKEN` env var
- [ ] Chat ID from `TELEGRAM_CHAT_ID` env var
- [ ] HTTP client with 10s timeout; retries on 429 (rate limit) with backoff
- [ ] Telegram API errors logged; message stays in SQS for retry (Lambda returns failure)

**Test Plan:**
- Unit: Mock HTTP client; verify request body sent to `https://api.telegram.org/bot<token>/sendMessage`
- Unit: 429 response → retry with backoff; 400 response → log and fail
- Integration: Send test message to a real test chat (manual, not automated)

**Terraform:**
- Notifier Lambda env vars: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`
- Notifier IAM role: CloudWatch Logs + SQS ReceiveMessage/DeleteMessage on `deal-notifications`

---

#### US-3.3: Format deal notification message

| Field | Value |
|-------|-------|
| Priority | Medium |
| Estimate | S |
| Dependencies | US-3.2 |

**As** a user,
**I want** notifications to be well-formatted and scannable,
**So that** I can quickly decide whether to act on a deal.

**Acceptance Criteria:**
- [ ] Uses Telegram HTML parse mode
- [ ] Format:
  ```
  <b>🔥 Good Deal Found</b>

  <b>{heading}</b>
  💰 {price} {currency}
  📍 {location}
  🔍 Query: {query}

  {reason}

  <a href="{canonicalUrl}">View on DBA.dk</a>
  ```
- [ ] Long reasons truncated to 200 chars
- [ ] Special characters escaped for HTML parse mode

**Test Plan:**
- Unit: Given a `DealNotification`, assert formatted string matches expected output
- Unit: Special chars in heading (`<`, `>`, `&`) are escaped
- Unit: Reason > 200 chars is truncated with `...`

---

### Epic 4: Infrastructure (Terraform)

---

#### US-4.1: Create SQS queues with dead-letter queues

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | None |

**As** an operator,
**I want** SQS queues with DLQs for each pipeline stage,
**So that** failed messages are preserved for debugging without blocking processing.

**Acceptance Criteria:**
- [ ] `dba-events` queue: visibility timeout 120s (4x Analyzer timeout), retention 1 day
- [ ] `dba-events-dlq`: max receive count 3, retention 7 days
- [ ] `deal-notifications` queue: visibility timeout 40s (4x Notifier timeout), retention 1 day
- [ ] `deal-notifications-dlq`: max receive count 3, retention 7 days
- [ ] All Standard queues (not FIFO — cheaper, sufficient for this use case)

**Test Plan:**
- `terraform plan` shows 4 queue resources + 2 redrive policies
- After apply: `aws sqs list-queues` shows all 4 queues
- Send a test message to each queue and receive it back

---

#### US-4.2: Create DynamoDB dedup table

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | S |
| Dependencies | None |

**As** the Analyzer service,
**I want** a DynamoDB table for event deduplication,
**So that** I can check processing history in <5ms per lookup.

**Acceptance Criteria:**
- [ ] Table name: `dba-analyzed-events`
- [ ] Partition key: `eventId` (String)
- [ ] TTL attribute: `expiresAt` (enabled)
- [ ] Billing: PAY_PER_REQUEST (no minimum cost)
- [ ] No sort key (simple key-value lookups only)

**Test Plan:**
- `terraform plan` shows table resource with correct config
- After apply: `aws dynamodb describe-table` confirms settings

---

#### US-4.3: IAM roles with least-privilege permissions

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-4.1, US-4.2 |

**As** a security-conscious operator,
**I want** each Lambda to have only the permissions it needs,
**So that** a compromise in one service limits blast radius.

**Acceptance Criteria:**
- [ ] **Ingestor role**: `AWSLambdaBasicExecutionRole` + SQS:SendMessage on `dba-events` + S3 access (existing)
- [ ] **Analyzer role**: `AWSLambdaBasicExecutionRole` + SQS:ReceiveMessage/DeleteMessage/GetQueueAttributes on `dba-events` + SQS:SendMessage on `deal-notifications` + DynamoDB:GetItem/PutItem on dedup table + Bedrock:InvokeModel on Nova 2 Lite
- [ ] **Notifier role**: `AWSLambdaBasicExecutionRole` + SQS:ReceiveMessage/DeleteMessage/GetQueueAttributes on `deal-notifications`
- [ ] Each role is a separate `aws_iam_role` resource (no shared roles)

**Test Plan:**
- `terraform plan` shows 3 role resources with distinct policies
- After deploy: attempt unauthorized action from each Lambda (e.g., Notifier calling Bedrock) → Access Denied

---

#### US-4.4: Deploy new Lambda functions with cost-optimized settings

| Field | Value |
|-------|-------|
| Priority | High |
| Estimate | M |
| Dependencies | US-2.1, US-3.1, US-4.3 |

**As** a developer,
**I want** all three Lambdas deployable via `terraform apply`,
**So that** the full pipeline can be provisioned in one command.

**Acceptance Criteria:**
- [ ] **Ingestor** (`dba-ingestor`): Java 25, 512MB, 30s timeout, SnapStart enabled
- [ ] **Analyzer** (`dba-analyzer`): `provided.al2023`, ARM64, 128MB, 30s timeout
- [ ] **Notifier** (`dba-telegram-notifier`): `provided.al2023`, ARM64, 128MB, 10s timeout
- [ ] All deployed from S3 (zip artifacts)
- [ ] Environment variables wired for queue URLs, table names, secrets
- [ ] Event source mappings: `dba-events` → Analyzer (batch 5), `deal-notifications` → Notifier (batch 1)

**Test Plan:**
- `terraform plan` is clean (no errors)
- After apply: invoke Ingestor manually → messages flow through pipeline → Telegram notification received

---

## 5. Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language (Analyzer) | Go | ~50ms cold start, tiny binary, excellent AWS SDK v2, easy to test locally |
| Language (Notifier) | Go | Same advantages; consistency with Analyzer |
| Language (Ingestor) | Java/Quarkus | Already built; refactor is cheaper than rewrite |
| Architecture (Go) | ARM64 + `provided.al2023` | Graviton is cheaper per-ms than x86; custom runtime required for Go |
| Memory (Go Lambdas) | 128MB | Go binaries use ~20-40MB; 128 is minimum and sufficient |
| Dedup storage | DynamoDB (On-Demand) | O(1) lookups, TTL auto-cleanup, $0 at rest, pennies at low volume |
| Queue type | SQS Standard | Free tier (1M req/month), native Lambda trigger, DLQ support; FIFO unnecessary |
| LLM model | Amazon Nova 2 Lite | Cheapest Bedrock model, sufficient for deal evaluation |
| Batch (Analyzer) | 5 | Balances throughput vs. timeout; keeps invocation well under 30s |
| Batch (Notifier) | 1 | Immediate delivery; deals are time-sensitive |

---

## 6. Message Schemas

### Queue: `dba-events` (Ingestor → Analyzer)

```json
{
  "eventId": "12345678",
  "adId": 12345678,
  "heading": "iPhone 15 Pro 256GB",
  "price": {
    "amount": 4500,
    "currencyCode": "DKK"
  },
  "location": "Kobenhavn",
  "canonicalUrl": "https://www.dba.dk/...",
  "imageUrls": ["https://..."],
  "brand": "Apple",
  "timestamp": 1719000000000,
  "extras": [{"id": "condition", "label": "Stand", "values": ["Brugt"]}],
  "query": "iphone 15 pro"
}
```

### Queue: `deal-notifications` (Analyzer → Notifier)

```json
{
  "eventId": "12345678",
  "heading": "iPhone 15 Pro 256GB",
  "price": {
    "amount": 4500,
    "currencyCode": "DKK"
  },
  "canonicalUrl": "https://www.dba.dk/...",
  "location": "Kobenhavn",
  "verdict": "GOOD_DEAL",
  "reason": "Priced at 4500 DKK, roughly 40% below typical used market price of 7500 DKK",
  "query": "iphone 15 pro"
}
```

---

## 7. Implementation Order

| Phase | Stories | Rationale |
|-------|---------|-----------|
| 1 | US-4.1, US-4.2 | Infrastructure must exist before services can use it |
| 2 | US-2.1, US-2.2, US-2.3, US-2.4, US-2.5, US-2.6 | Analyzer is the core value — build it first |
| 3 | US-3.1, US-3.2, US-3.3 | Notifier delivers value to user |
| 4 | US-1.1, US-1.2 | Refactor Ingestor last (requires queue) |
| 5 | US-4.3, US-4.4, US-1.3 | Wire IAM, deploy all, schedule |
| 6 | E2E verification | Full pipeline smoke test |

---

## 8. Verification & Testing Strategy

### Per-Service Testing

| Service | Unit Tests | Integration Tests | Tools |
|---------|-----------|-------------------|-------|
| Ingestor | Mock SQS client, verify message format | Quarkus DevServices + LocalStack | `./mvnw test` |
| Analyzer | Mock DynamoDB/Bedrock/SQS, test all paths | LocalStack (SQS + DynamoDB) | `go test ./...` |
| Notifier | Mock HTTP client, verify Telegram payload | Real Telegram test bot (manual) | `go test ./...` |

### End-to-End Verification

1. Deploy all infrastructure with `terraform apply`
2. Manually invoke Ingestor: `aws lambda invoke --function-name dba-ingestor --payload '{"detail":{"queries":["test"],"limit":5}}'`
3. Check `dba-events` queue: `aws sqs get-queue-attributes --queue-url ... --attribute-names ApproximateNumberOfMessages`
4. Wait for Analyzer to trigger → check DynamoDB for new items
5. If good deal found → confirm Telegram message received
6. Check DLQs are empty (no failures)

### Contract Testing

JSON fixtures shared between services ensure schema compatibility:
- `testdata/dba-event.json` — valid Ingestor output / Analyzer input
- `testdata/deal-notification.json` — valid Analyzer output / Notifier input
- Each service's tests load and deserialize the other's fixture to verify compatibility

---

## 9. Project Structure (Final State)

```
services/
  dba/                    # DBA Ingestor (Java/Quarkus) — refactored
  analyzer/               # Deal Analyzer (Go)
    cmd/
      analyzer/
        main.go
    internal/
      handler/
      dedup/
      llm/
      publisher/
    go.mod
    go.sum
    Makefile
  telegram-notifier/      # Telegram Notifier (Go)
    cmd/
      notifier/
        main.go
    internal/
      handler/
      telegram/
    go.mod
    go.sum
    Makefile
terraform/
  main.tf
  lambda.tf               # 3 Lambda functions + event source mappings
  sqs.tf                  # Queues + DLQs + redrive policies
  dynamodb.tf             # Dedup table
  eventbridge.tf          # Schedule rule
  iam.tf                  # 3 IAM roles (extracted from lambda.tf)
  s3.tf                   # Deployment artifacts
  variables.tf            # telegram_bot_token, telegram_chat_id, etc.
```

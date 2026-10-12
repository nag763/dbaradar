# dbaradar

An automated, serverless event-driven radar for Danish classifieds ([DBA.dk](https://www.dba.dk)). It regularly polls user-defined marketplace searches, evaluates listings with Amazon Bedrock GenAI (Amazon Nova Lite) to detect genuine bargains, and instantly delivers alerts and interactive subscription control via Telegram.

---

## Architecture Overview

```
                        ┌──────────────────┐
                        │   EventBridge    │
                        │  (Cron Trigger)  │
                        └────────┬─────────┘
                                 │
                                 ▼
                     ┌───────────────────────┐
                     │     dba-ingestor      │ ◄── Reads queries from DynamoDB
                     │    (Go Lambda / AL23) │ ──► Queries DBA.dk API
                     └───────────┬───────────┘
                                 │ SQS (dba-events)
                                 ▼
                     ┌───────────────────────┐
                     │     dba-analyzer      │ ◄── DynamoDB (Deduplication)
                     │   (Go Lambda / AL23)  │ ◄── Bedrock Nova 2 Lite (LLM Evaluation)
                     └───────────┬───────────┘
                                 │ SQS (deal-notifications)
                                 ▼
                     ┌───────────────────────┐
                     │ dba-telegram-notifier │
                     │   (Go Lambda / AL23)  │ ──► Telegram Bot API (Alerts)
                     └───────────────────────┘

Interactive Bot Subscription Flow:
Telegram User ──► API Gateway HTTP API v2 ──► dba-authorizer (Rust Lambda)
                                                      │ (Validates Telegram IP & Secret)
                                                      ▼
                                              dba-webhooks (Go Lambda)
                                                      │
                                                      ▼
                                       DynamoDB (trusted-chat-queries)
```

### Services

| Service | Directory | Tech Stack | Role |
| :--- | :--- | :--- | :--- |
| **Ingestor** | [`services/dba`](file:///home/usr/workspace/dbaradar/services/dba) | Go (ARM64) | Triggered by EventBridge; reads active search queries from DynamoDB, fetches matching ads from DBA in parallel, and pushes raw listings to SQS (`dba-events`). |
| **Analyzer** | [`services/analyzer`](file:///home/usr/workspace/dbaradar/services/analyzer) | Go (ARM64) | Consumes SQS batch; skips already-processed ads via DynamoDB cache, invokes AWS Bedrock (Amazon Nova 2 Lite) for deal analysis, and forwards bargains (`GOOD_DEAL`) to `deal-notifications`. |
| **Telegram Notifier** | [`services/telegram-notifier`](file:///home/usr/workspace/dbaradar/services/telegram-notifier) | Go (ARM64) | Consumes good deal notifications and dispatches formatted HTML alerts via the Telegram Bot API. |
| **Webhooks** | [`services/webhooks`](file:///home/usr/workspace/dbaradar/services/webhooks) | Go (ARM64) | Handles Telegram webhook interactions (`/add`, `/remove`, `/list`, `/clear`) to allow users to manage subscriptions directly from Telegram chat. |
| **Authorizer** | [`services/authorizer`](file:///home/usr/workspace/dbaradar/services/authorizer) | Rust (Cargo Lambda) | Custom API Gateway authorizer verifying Telegram secret token headers and official Telegram IP ranges. |

---

## Features

- **Automated GenAI Deal Evaluation**: Evaluates product condition, accessories, pricing context, and search queries against market averages using Amazon Bedrock (`eu.amazon.nova-2-lite-v1:0`).
- **Cost-Optimized Serverless Architecture**: Built on AWS Lambda (ARM64 / Graviton), SQS standard queues, DynamoDB On-Demand (pay-per-request), and pay-per-token Bedrock inference.
- **Deduplication with TTL**: DynamoDB stores evaluated ad IDs with automatic TTL expiration to prevent redundant LLM invocations and alert spam.
- **Interactive Telegram Bot**: Manage search subscriptions straight from Telegram using `/add <query>`, `/remove <query>`, `/list`, and `/clear`.
- **Hardened Webhook Security**: Multi-tier authentication with a dedicated Rust Lambda authorizer validating Telegram IP CIDR blocks and webhook secret tokens before reaching application logic.

---

## Repository Structure

```
.
├── services/
│   ├── dba/                  # DBA ingestor / ad poller
│   ├── analyzer/             # Bedrock LLM deal evaluator & deduplication
│   ├── telegram-notifier/    # Telegram alert dispatcher
│   ├── webhooks/             # Telegram command webhook handler
│   └── authorizer/           # Rust API Gateway custom authorizer
├── terraform/                # Infrastructure as Code (AWS eu-central-1)
├── docs/                     # Design documents & feature specifications
├── Makefile                  # Monorepo build and test orchestrator
└── README.md
```

---

## Prerequisites

- **Go**: `1.26+`
- **Rust & Cargo Lambda**: For building `services/authorizer`
- **AWS CLI**: Configured with credentials for deployment
- **Terraform**: `>= 1.15`
- **Telegram Bot Token**: Created via [@BotFather](https://t.me/BotFather)

---

## Building & Testing

Use the root [`Makefile`](file:///home/usr/workspace/dbaradar/Makefile) to build or test all services at once:

```bash
# Build all Lambda deployment packages
make build

# Run unit tests across all services
make test

# Clean build artifacts
make clean
```

Or work on individual services:

```bash
# Analyzer tests
cd services/analyzer && go test ./...

# Webhooks tests (with race detection)
cd services/webhooks && go test -v -race ./...

# Authorizer tests
cd services/authorizer && cargo test
```

---

## Deployment Guide

### 1. Build the Lambda Binaries
Compile all Lambda artifacts:
```bash
make build
```

### 2. Configure Terraform Variables
Create a `terraform/terraform.tfvars` file based on [`terraform/terraform.tfvars.example`](file:///home/usr/workspace/dbaradar/terraform/terraform.tfvars.example):

```hcl
telegram_bot_token      = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
telegram_chat_id        = "your-default-chat-id"
telegram_webhook_secret = "your-custom-webhook-secret-token"
trusted_chat_ids        = ["your-default-chat-id"]
```

> [!CAUTION]
> Never commit `terraform.tfvars` or `*.tfstate` to source control. They are ignored by `.gitignore`.

### 3. Deploy Infrastructure
Verify validity and plan changes:
```bash
cd terraform
terraform init
terraform fmt
terraform validate
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

### 4. Register Webhook with Telegram
After deployment, retrieve the generated webhook URL from the outputs and register it:
```bash
WEBHOOK_URL=$(terraform output -raw webhook_url)

curl -F "url=${WEBHOOK_URL}" \
     -F "secret_token=<YOUR_TELEGRAM_WEBHOOK_SECRET>" \
     https://api.telegram.org/bot<YOUR_TELEGRAM_BOT_TOKEN>/setWebhook
```

---

## Bot Usage

Once deployed, start chatting with your bot on Telegram:

- `/add <query>` &mdash; Subscribe to DBA searches (e.g. `/add thinkpad x1 carbon` or `/add iphone 15 pro`)
- `/list` &mdash; List your active search subscriptions
- `/remove <query>` &mdash; Stop tracking a specific search query
- `/clear` &mdash; Remove all queries for your chat

When an ad matching your queries appears on DBA.dk and is deemed a bargain by the LLM, you will receive an alert with price details, market assessment, and a direct link to the listing.

---

## License

This project is licensed under the MIT License.

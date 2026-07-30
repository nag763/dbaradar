# Telegram Webhooks Lambda Service

A Go Lambda service that processes Telegram webhook events for the dbaradar deal analyzer. Handles bot commands (`/add`, `/remove`, `/clear`, `/list`) for user subscriptions with rate limiting and token-based authentication.

## Architecture

```
Telegram Bot API
    ↓
API Gateway HTTP API v2
    ├─ Rate limiting (50 req/min per IP)
    └─ Header validation
    ↓
Lambda Function (dba-webhooks)
    ├─ Rate limiter middleware (chi/httprate)
    ├─ Token verification middleware
    ├─ Chi v5 router
    └─ Command handlers (echo in Phase 1)
```

## Supported Commands

- **`/add <query>`** - Add a subscription (Phase 1: echoes back)
- **`/remove <query>`** - Remove a subscription (Phase 1: echoes back)
- **`/clear`** - Clear all subscriptions (Phase 1: echoes back)
- **`/list`** - List all subscriptions (Phase 1: echoes back)

## Configuration

The service uses two-tier configuration resolution:

1. **SSM Parameter (preferred)**: Set `TELEGRAM_SECRET_TOKEN_PARAM` env var to the SSM parameter name
2. **Direct env var (fallback)**: Set `TELEGRAM_SECRET_TOKEN` with the actual token value

Example (Lambda environment):
```
TELEGRAM_SECRET_TOKEN_PARAM = "/dba-webhooks/telegram_secret_token"
```

## Endpoints

### POST /webhook
Processes Telegram webhook events. Requires `X-Telegram-Bot-Api-Secret-Token` header.

**Request:**
```bash
curl -X POST https://{api_id}.execute-api.eu-central-1.amazonaws.com/webhook \
  -H "X-Telegram-Bot-Api-Secret-Token: <secret>" \
  -H "Content-Type: application/json" \
  -d '{
    "update_id": 123,
    "message": {
      "message_id": 1,
      "from": {"id": 456, "first_name": "User"},
      "chat": {"id": 789, "type": "private"},
      "date": 1627000000,
      "text": "/add iphone 15"
    }
  }'
```

**Response (200 OK):**
```json
{
  "method": "sendMessage",
  "chat_id": 789,
  "text": "Echo /add with args: [iphone 15]"
}
```

### GET /health
Health check endpoint. No authentication required.

**Response (200 OK):**
```json
{
  "status": "healthy"
}
```

## Rate Limiting

- **API Gateway Level**: 50 requests per minute per IP (coarse-grained)
- **Lambda Level**: 50 requests per minute per IP (fine-grained via chi/httprate)

The health endpoint is still rate-limited but bypasses token verification.

## Development

### Build

```bash
cd services/webhooks
make build
```

Creates:
- `build/bootstrap` - ARM64 Linux binary
- `build/webhooks.zip` - Deployment package

### Test

```bash
cd services/webhooks
make test
```

Runs all unit tests with race detector.

### Local Testing

Set environment variables:
```bash
export TELEGRAM_SECRET_TOKEN="test-secret"
```

Then run tests or build.

## Deployment

1. **Build the service:**
   ```bash
   cd services/webhooks
   make build
   ```

2. **Plan Terraform changes:**
   ```bash
   cd terraform
   terraform plan -var-file=terraform.tfvars
   ```

3. **Apply Terraform (manual approval):**
   ```bash
   terraform apply -var-file=terraform.tfvars
   ```

   Set `telegram_webhook_secret` in tfvars (keep secret).

4. **Get webhook URL:**
   ```bash
   terraform output webhook_url
   ```

5. **Register with Telegram:**
   ```bash
   curl -F "url=<webhook_url>" \
        -F "secret_token=<secret>" \
        https://api.telegram.org/bot<token>/setWebhook
   ```

6. **Verify:**
   - Send `/add test` to the bot
   - Expect echo response: "Echo /add with args: [test]"

## Security

- **Token Storage**: Encrypted in AWS SSM as SecureString
- **Token Validation**: Two-layer (API Gateway + Lambda)
- **Logging**: Structured JSON, no PII or sensitive data
- **Rate Limiting**: Per-IP throttling prevents abuse
- **Input Validation**: JSON parsing validates Telegram update format

## Monitoring

CloudWatch Logs:
- Log Group: `/aws/lambda/dba-webhooks`
- Retention: 14 days
- Format: Structured JSON

API Gateway access logs available in `/aws/apigateway/dba-webhooks`.

## Future Extensions

### Phase 2: DynamoDB Integration
- Connect to `trusted_chat_queries` table
- Implement actual CRUD operations
- Store user subscriptions

### Phase 3: Chat ID Validation
- Validate sender against `TRUSTED_CHAT_IDS_PARAM`
- Reject commands from untrusted users

### Phase 4: Enhanced Command Parsing
- Support quoted arguments: `/add "iphone 15 pro"`
- Support key=value pairs: `/add query="iphone" min_price=1000`
- Add argument validation

### Phase 5: Rich Responses
- HTML formatting (bold, links, code)
- Inline keyboards for confirmations
- Message editing for updates
- Image/preview support

## Dependencies

- Go 1.26+
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/httprate` - Rate limiting
- `github.com/aws/aws-lambda-go` - Lambda runtime
- `github.com/aws/aws-sdk-go-v2` - AWS SDK

## Files

```
services/webhooks/
├── cmd/main.go                         # Lambda entrypoint
├── internal/
│   ├── config/
│   │   ├── config.go                   # SSM config resolution
│   │   └── config_test.go
│   ├── middleware/
│   │   ├── auth.go                     # Token verification
│   │   ├── auth_test.go
│   ├── handler/
│   │   ├── handler.go                  # HTTP handlers
│   │   └── handler_test.go
│   ├── router/
│   │   ├── router.go                   # Chi router setup
│   │   └── router_test.go
│   └── model/
│       ├── telegram.go                 # Telegram data models
│       ├── command.go                  # Command parser
│       └── command_test.go
├── Makefile                            # Build automation
├── go.mod / go.sum                     # Go dependencies
└── README.md                           # This file
```

## References

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Chi v5 Documentation](https://github.com/go-chi/chi)
- [AWS Lambda Go](https://github.com/aws/aws-lambda-go)
- [Chi HTTP Rate](https://github.com/go-chi/httprate)

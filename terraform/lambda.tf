# Lambda functions for the deal analyzer pipeline.

# --- DBA Ingestor (Quarkus Native) ---

resource "aws_lambda_function" "dba_ingestor" {
  function_name    = "dba-ingestor"
  role             = aws_iam_role.ingestor.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["x86_64"]
  s3_bucket        = aws_s3_object.ingestor_code.bucket
  s3_key           = aws_s3_object.ingestor_code.key
  source_code_hash = filebase64sha256(local.ingestor_zip_path)
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      DBA_EVENTS_QUEUE_URL       = aws_sqs_queue.dba_events.url,
      TRUSTED_CHAT_QUERIES_TABLE = aws_dynamodb_table.trusted_chat_queries.name,
      TRUSTED_CHAT_IDS_PARAM     = aws_ssm_parameter.trusted_chat_ids.name,
    }
  }
}

# --- Analyzer (Go, ARM64) ---

resource "aws_lambda_function" "analyzer" {
  function_name    = "dba-analyzer"
  role             = aws_iam_role.analyzer.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  s3_bucket        = aws_s3_object.analyzer_code.bucket
  s3_key           = aws_s3_object.analyzer_code.key
  source_code_hash = filebase64sha256(local.analyzer_zip_path)
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      DEDUP_TABLE_NAME             = aws_dynamodb_table.analyzed_events.name
      DEAL_NOTIFICATIONS_QUEUE_URL = aws_sqs_queue.deal_notifications.url
      TRUSTED_CHAT_IDS_PARAM       = aws_ssm_parameter.trusted_chat_ids.name
      TRUSTED_CHAT_QUERIES_TABLE   = aws_dynamodb_table.trusted_chat_queries.name
    }
  }
}

resource "aws_lambda_event_source_mapping" "analyzer_trigger" {
  event_source_arn                   = aws_sqs_queue.dba_events.arn
  function_name                      = aws_lambda_function.analyzer.arn
  batch_size                         = 5
  enabled                            = true
  function_response_types            = ["ReportBatchItemFailures"]
  maximum_batching_window_in_seconds = 10
}

# --- Telegram Notifier (Go, ARM64) ---

resource "aws_ssm_parameter" "telegram_bot_token" {
  name        = "/dba-notifier/telegram_bot_token"
  description = "Telegram Bot API token for deal notifications"
  type        = "SecureString"
  value       = var.telegram_bot_token
}

resource "aws_lambda_function" "telegram_notifier" {
  function_name    = "dba-telegram-notifier"
  role             = aws_iam_role.telegram_notifier.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  s3_bucket        = aws_s3_object.notifier_code.bucket
  s3_key           = aws_s3_object.notifier_code.key
  source_code_hash = filebase64sha256(local.notifier_zip_path)
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      TELEGRAM_BOT_TOKEN_PARAM = aws_ssm_parameter.telegram_bot_token.name
      TELEGRAM_CHAT_ID         = var.telegram_chat_id
    }
  }
}

resource "aws_cloudwatch_log_group" "dba_ingestor" {
  name              = "/aws/lambda/dba-ingestor"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "analyzer" {
  name              = "/aws/lambda/dba-analyzer"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "telegram_notifier" {
  name              = "/aws/lambda/dba-telegram-notifier"
  retention_in_days = 14
}

resource "aws_lambda_event_source_mapping" "notifier_trigger" {
  event_source_arn = aws_sqs_queue.deal_notifications.arn
  function_name    = aws_lambda_function.telegram_notifier.arn
  batch_size       = 1
  enabled          = true
}

# --- Telegram Webhooks (Go, ARM64) ---

resource "aws_ssm_parameter" "telegram_webhook_secret" {
  name        = "/dba-webhooks/telegram_secret_token"
  description = "Secret token for Telegram webhook verification"
  type        = "SecureString"
  value       = var.telegram_webhook_secret
}

resource "aws_lambda_function" "webhooks" {
  function_name    = "dba-webhooks"
  role             = aws_iam_role.webhooks.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  s3_bucket        = aws_s3_object.webhooks_code.bucket
  s3_key           = aws_s3_object.webhooks_code.key
  source_code_hash = filebase64sha256(local.webhooks_zip_path)
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      TELEGRAM_SECRET_TOKEN_PARAM = aws_ssm_parameter.telegram_webhook_secret.name
      TRUSTED_CHAT_IDS_PARAM      = aws_ssm_parameter.trusted_chat_ids.name
      TELEGRAM_IP_RANGES_PARAM    = aws_ssm_parameter.telegram_ip_ranges.name
    }
  }
}

resource "aws_cloudwatch_log_group" "webhooks" {
  name              = "/aws/lambda/dba-webhooks"
  retention_in_days = 14
}

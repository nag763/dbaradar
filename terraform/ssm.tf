# SSM parameter holding the comma-separated list of trusted chat IDs from trusted_chat_subscriptions.
resource "aws_ssm_parameter" "trusted_chat_ids" {
  name  = "/dbanalyzer/trusted_chat_ids"
  type  = "String"
  value = join(",", keys(var.trusted_chat_subscriptions))
  tags  = var.default_tags
}

# SSM parameter holding the comma-separated list of Telegram official IP ranges for webhook validation.
resource "aws_ssm_parameter" "telegram_ip_ranges" {
  name  = "/dbanalyzer/telegram_ip_ranges"
  type  = "String"
  value = join(",", var.telegram_ip_ranges)
  tags  = var.default_tags
}

# --- Telegram Webhooks (Go, ARM64) ---

resource "aws_ssm_parameter" "telegram_webhook_secret" {
  name        = "/dba-webhooks/telegram_secret_token"
  description = "Secret token for Telegram webhook verification"
  type        = "SecureString"
  value       = var.telegram_webhook_secret
}

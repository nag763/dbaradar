variable "telegram_bot_token" {
  description = "Telegram Bot API token for the notifier Lambda"
  type        = string
  sensitive   = true
}

variable "telegram_chat_id" {
  description = "Telegram chat ID to send deal notifications to"
  type        = string
}

variable "telegram_webhook_secret" {
  description = "Secret token for Telegram webhook verification"
  type        = string
  sensitive   = true
}

variable "ingestor_schedule_daytime" {
  description = "EventBridge cron expression for daytime triggers (every 30 min, 07:00-21:59 UTC = 8 AM-10 PM Paris)"
  type        = string
  default     = "cron(0,30 7-21 * * ? *)"
}

variable "ingestor_schedule_offpeak" {
  description = "EventBridge cron expression for off-peak triggers (every hour, 22:00-23:59 and 00:00-06:59 UTC)"
  type        = string
  default     = "cron(0 0-6,22-23 * * ? *)"
}

variable "default_tags" {
  description = "Default tags applied to all AWS resources created by Terraform"
  type        = map(string)
  default = {
    Project     = "dbaradar"
    ManagedBy   = "Terraform"
    Environment = "production"
  }
}

variable "trusted_chat_subscriptions" {
  description = "Mapping of chatId -> list of queries to populate trusted-chat-queries"
  type        = map(list(string))
  default = {
    "5253905011" = ["canon r7", "canon r8", "canon r6", "olympus om-d em-5", "lumix gx80", "sony a6700", "sony a6600", "sony a6400", "canon r10", "nikon z6", "nikon z5", "olympus om-d e-m10"]
  }
}

variable "telegram_ip_ranges" {
  description = "Telegram official IP ranges for webhook validation"
  type        = list(string)
  default     = ["149.154.160.0/20", "91.108.4.0/22"]
}

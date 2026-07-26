variable "telegram_bot_token" {
  description = "Telegram Bot API token for the notifier Lambda"
  type        = string
  sensitive   = true
}

variable "telegram_chat_id" {
  description = "Telegram chat ID to send deal notifications to"
  type        = string
}

variable "dba_search_queries" {
  description = "List of search queries for the DBA ingestor"
  type        = list(string)
  default     = ["canon r7", "canon r8", "canon r6", "olympus om-d em-5", "lumix gx80", "sony a6700", "sony a6600", "sony a6400", "canon r10", "nikon z6", "nikon z5", "olympus om-d e-m10"]
}

variable "ingestor_schedule_daytime" {
  description = "EventBridge cron expression for daytime triggers (every 10 min, 09:00-21:59 UTC)"
  type        = string
  default     = "cron(*/15 9-21 * * ? *)"
}

variable "ingestor_schedule_offpeak" {
  description = "EventBridge cron expression for off-peak triggers (every 30 min, 22:00-08:59 UTC)"
  type        = string
  default     = "cron(0,30 0-8,22-23 * * ? *)"
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

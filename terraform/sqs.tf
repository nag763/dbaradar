# SQS queues connecting the pipeline stages.
# Standard queues (not FIFO) — cheaper and sufficient for this use case.

# --- dba-events: Ingestor -> Analyzer ---

resource "aws_sqs_queue" "dba_events_dlq" {
  name                      = "dba-events-dlq"
  message_retention_seconds = 604800 # 7 days
}

resource "aws_sqs_queue" "dba_events" {
  name                       = "dba-events"
  visibility_timeout_seconds = 120   # 4x Analyzer Lambda timeout (30s)
  message_retention_seconds  = 86400 # 1 day

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dba_events_dlq.arn
    maxReceiveCount     = 3
  })
}

# --- deal-notifications: Analyzer -> Telegram Notifier ---

resource "aws_sqs_queue" "deal_notifications_dlq" {
  name                      = "deal-notifications-dlq"
  message_retention_seconds = 604800 # 7 days
}

resource "aws_sqs_queue" "deal_notifications" {
  name                       = "deal-notifications"
  visibility_timeout_seconds = 40    # 4x Notifier Lambda timeout (10s)
  message_retention_seconds  = 86400 # 1 day

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.deal_notifications_dlq.arn
    maxReceiveCount     = 3
  })
}

# DynamoDB table for deduplicating analyzed DBA events.
# PAY_PER_REQUEST keeps costs at zero when idle.

resource "aws_dynamodb_table" "analyzed_events" {
  name         = "dba-analyzed-events"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }
}

# DynamoDB table for mapping trusted chat IDs to queries.
resource "aws_dynamodb_table" "trusted_chat_queries" {
  name         = "trusted-chat-queries"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "chatId"
  range_key    = "query"

  attribute {
    name = "chatId"
    type = "S"
  }

  attribute {
    name = "query"
    type = "S"
  }

  # Global secondary index to look up chatIds by query quickly
  global_secondary_index {
    name            = "query-to-chatId-gsi"
    hash_key        = "query"
    projection_type = "ALL"
  }
}

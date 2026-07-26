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

# Populate trusted-chat-queries DynamoDB table with items from var.trusted_chat_subscriptions.

locals {
  effective_subscriptions = var.trusted_chat_subscriptions

  # flatten to a list of objects { chatId, query, id }
  trusted_pairs = flatten([
    for cid, queries in local.effective_subscriptions : [
      for q in queries : {
        chatId = cid
        query  = q
        id     = cid == "" ? q : "${cid}::${replace(q, " ", "_")}" # deterministic id
      }
    ]
  ])

  trusted_pairs_map = { for p in local.trusted_pairs : p.id => p }
}

resource "aws_dynamodb_table_item" "trusted_chat_query_items" {
  for_each = local.trusted_pairs_map

  table_name = aws_dynamodb_table.trusted_chat_queries.name
  hash_key   = "chatId"
  range_key  = "query"

  item = jsonencode({
    chatId = { S = each.value.chatId }
    query  = { S = each.value.query }
  })
}

# SSM parameter holding the comma-separated list of trusted chat IDs from trusted_chat_subscriptions.
resource "aws_ssm_parameter" "trusted_chat_ids" {
  name  = "/dbanalyzer/trusted_chat_ids"
  type  = "String"
  value = join(",", keys(var.trusted_chat_subscriptions))
  tags  = var.default_tags
}

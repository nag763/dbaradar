# EventBridge scheduled rules triggering the DBA ingestor.

# 1. Daytime schedule: Every 10 minutes between 09:00 and 21:59 (9 to 22)
resource "aws_cloudwatch_event_rule" "dba_schedule_daytime" {
  name                = "dba-ingestor-schedule-daytime"
  description         = "Triggers DBA ingestor Lambda every 10 minutes between 09:00 and 22:00"
  schedule_expression = var.ingestor_schedule_daytime
}

resource "aws_cloudwatch_event_target" "dba_ingestor_daytime" {
  rule      = aws_cloudwatch_event_rule.dba_schedule_daytime.name
  target_id = "dba-ingestor-daytime"
  arn       = aws_lambda_function.dba_ingestor.arn
}

resource "aws_lambda_permission" "eventbridge_invoke_ingestor_daytime" {
  statement_id  = "AllowEventBridgeInvokeDaytime"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.dba_ingestor.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.dba_schedule_daytime.arn
}

# 2. Off-peak schedule: Every 30 minutes outside daytime hours (22:00 to 08:59)
resource "aws_cloudwatch_event_rule" "dba_schedule_offpeak" {
  name                = "dba-ingestor-schedule-offpeak"
  description         = "Triggers DBA ingestor Lambda every 30 minutes outside 09:00 to 22:00"
  schedule_expression = var.ingestor_schedule_offpeak
}

resource "aws_cloudwatch_event_target" "dba_ingestor_offpeak" {
  rule      = aws_cloudwatch_event_rule.dba_schedule_offpeak.name
  target_id = "dba-ingestor-offpeak"
  arn       = aws_lambda_function.dba_ingestor.arn
}

resource "aws_lambda_permission" "eventbridge_invoke_ingestor_offpeak" {
  statement_id  = "AllowEventBridgeInvokeOffpeak"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.dba_ingestor.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.dba_schedule_offpeak.arn
}

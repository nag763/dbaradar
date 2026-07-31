resource "random_string" "random_endpoint" {
  length  = 96
  special = false
  lower   = true
  upper   = true
}

resource "aws_apigatewayv2_api" "webhooks" {
  name          = "dba-webhooks-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["POST"]
    allow_headers = ["X-Telegram-Bot-Api-Secret-Token", "Content-Type"]
  }

  tags = {
    Service = "dba-webhooks"
  }
}

resource "aws_apigatewayv2_stage" "webhooks" {
  api_id      = aws_apigatewayv2_api.webhooks.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.apigateway.arn

    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      resourcePath   = "$context.resourcePath"
      status         = "$context.status"
      protocol       = "$context.protocol"
      responseLength = "$context.responseLength"
      error          = "$context.error.message"
    })
  }

  tags = {
    Service = "dba-webhooks"
  }
}

resource "aws_cloudwatch_log_group" "apigateway" {
  name              = "/aws/apigateway/dba-webhooks"
  retention_in_days = 14

  tags = {
    Service = "dba-webhooks"
  }
}

resource "aws_apigatewayv2_integration" "lambda" {
  api_id                 = aws_apigatewayv2_api.webhooks.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.webhooks.invoke_arn
  payload_format_version = "2.0"

}

resource "aws_apigatewayv2_route" "webhook" {
  api_id             = aws_apigatewayv2_api.webhooks.id
  route_key          = "POST /${random_string.random_endpoint.result}"
  target             = "integrations/${aws_apigatewayv2_integration.lambda.id}"
  authorization_type = "CUSTOM"
  authorizer_id      = aws_apigatewayv2_authorizer.lambda_auth.id
}

resource "aws_lambda_permission" "apigateway_webhook" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.webhooks.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.webhooks.execution_arn}/*/*"

}

resource "aws_apigatewayv2_authorizer" "lambda_auth" {
  api_id                  = aws_apigatewayv2_api.webhooks.id
  authorizer_type         = "REQUEST"
  name                    = "trusted-caller-authorizer"
  authorizer_uri          = aws_lambda_function.authorizer.invoke_arn
  enable_simple_responses = true

  authorizer_payload_format_version = "2.0"
  authorizer_result_ttl_in_seconds  = 3600
  identity_sources                  = ["$request.header.X-Telegram-Bot-Api-Secret-Token", "$context.identity.sourceIp"]
}

resource "aws_lambda_permission" "authorizer" {
  statement_id  = "AllowAPIGatewayInvokeAuthorizer"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.authorizer.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_apigatewayv2_api.webhooks.execution_arn}/*/*"
}

output "webhook_url" {
  value       = "${aws_apigatewayv2_stage.webhooks.invoke_url}${random_string.random_endpoint.result}"
  description = "Webhook URL for Telegram bot"
}

# Least-privilege IAM roles for each Lambda in the pipeline.

# --- DBA Ingestor Role ---

resource "aws_iam_role" "ingestor" {
  name = "dba-ingestor-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  ]

  inline_policy {
    name = "ingestor_permissions"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect   = "Allow"
          Action   = ["sqs:SendMessage", "sqs:SendMessageBatch", "sqs:GetQueueUrl"]
          Resource = aws_sqs_queue.dba_events.arn
        },
        {
          Effect   = "Allow"
          Action   = ["s3:GetObject", "s3:PutObject"]
          Resource = "${aws_s3_bucket.lambda_bucket.arn}/*"
        }
      ]
    })
  }
}

# --- Analyzer Role ---

resource "aws_iam_role" "analyzer" {
  name = "dba-analyzer-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    "arn:aws:iam::aws:policy/service-role/AWSLambdaSQSQueueExecutionRole"
  ]

  inline_policy {
    name = "analyzer_permissions"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect   = "Allow"
          Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
          Resource = "${aws_cloudwatch_log_group.analyzer.arn}:*"
        },
        {
          Effect   = "Allow"
          Action   = ["sqs:SendMessage"]
          Resource = aws_sqs_queue.deal_notifications.arn
        },
        {
          Effect   = "Allow"
          Action   = ["dynamodb:GetItem", "dynamodb:BatchGetItem", "dynamodb:PutItem"]
          Resource = aws_dynamodb_table.analyzed_events.arn
        },
        {
          Effect = "Allow"
          Action = ["bedrock:InvokeModel"]
          Resource = [
            "arn:aws:bedrock:eu-central-1:${data.aws_caller_identity.current.account_id}:inference-profile/eu.amazon.nova-2-lite-v1:0",
            "arn:aws:bedrock:eu-*::foundation-model/amazon.nova-2-lite-v1:0"
          ]
        }
      ]
    })
  }
}

# --- Telegram Notifier Role ---

resource "aws_iam_role" "telegram_notifier" {
  name = "dba-telegram-notifier-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    "arn:aws:iam::aws:policy/service-role/AWSLambdaSQSQueueExecutionRole"
  ]

  inline_policy {
    name = "telegram_notifier_permissions"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect   = "Allow"
          Action   = ["logs:CreateLogStream", "logs:PutLogEvents"]
          Resource = "${aws_cloudwatch_log_group.telegram_notifier.arn}:*"
        },
        {
          Effect   = "Allow"
          Action   = ["ssm:GetParameter", "ssm:GetParameters"]
          Resource = aws_ssm_parameter.telegram_bot_token.arn
        }
      ]
    })
  }
}

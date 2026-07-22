
resource "aws_lambda_function" "app" {
  function_name = "dba-handler"
  role          = aws_iam_role.lambda_exec.arn
  handler       = "io.quarkus.amazon.lambda.runtime.QuarkusStreamHandler::handleRequest"
  runtime       = "java25"


  s3_bucket        = aws_s3_object.lambda_code.bucket
  s3_key           = aws_s3_object.lambda_code.key
  source_code_hash = filebase64sha256(local.zip_path)
  timeout          = 10
  memory_size      = 512

  snap_start {
    apply_on = "PublishedVersions"
  }
}

resource "aws_iam_role" "lambda_exec" {
  name = "lambda_s3_minimal_role"

  # 1. Trust Policy (Allows Lambda to assume the role)
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  # 2. Managed Policy for minimal CloudWatch logs
  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  ]

  # 3. Inline S3 Policy
  inline_policy {
    name = "s3_access"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject"]
        Resource = "${aws_s3_bucket.lambda_bucket.arn}/*"
      }]
    })
  }
}

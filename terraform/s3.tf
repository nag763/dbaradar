# S3 bucket and deployment artifacts for all Lambda functions.

locals {
  ingestor_zip_path = "../services/dba/target/function.zip"
  analyzer_zip_path = "../services/analyzer/build/analyzer.zip"
  notifier_zip_path = "../services/telegram-notifier/build/notifier.zip"
  webhooks_zip_path = "../services/webhooks/build/webhooks.zip"
}

resource "aws_s3_bucket" "lambda_bucket" {
  bucket        = "dbaradar"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "lambda_bucket" {
  bucket                  = aws_s3_bucket.lambda_bucket.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "lambda_bucket" {
  bucket = aws_s3_bucket.lambda_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "lambda_bucket" {
  bucket = aws_s3_bucket.lambda_bucket.id

  rule {
    id     = "expire-old-deployments"
    status = "Enabled"

    expiration {
      days = 30
    }
  }
}

resource "aws_s3_object" "ingestor_code" {
  bucket = aws_s3_bucket.lambda_bucket.bucket
  key    = "deployments/dba-ingestor-${filemd5(local.ingestor_zip_path)}.zip"
  source = local.ingestor_zip_path
  etag   = filemd5(local.ingestor_zip_path)
}

resource "aws_s3_object" "analyzer_code" {
  bucket = aws_s3_bucket.lambda_bucket.bucket
  key    = "deployments/dba-analyzer-${filemd5(local.analyzer_zip_path)}.zip"
  source = local.analyzer_zip_path
  etag   = filemd5(local.analyzer_zip_path)
}

resource "aws_s3_object" "notifier_code" {
  bucket = aws_s3_bucket.lambda_bucket.bucket
  key    = "deployments/dba-telegram-notifier-${filemd5(local.notifier_zip_path)}.zip"
  source = local.notifier_zip_path
  etag   = filemd5(local.notifier_zip_path)
}

resource "aws_s3_object" "webhooks_code" {
  bucket = aws_s3_bucket.lambda_bucket.bucket
  key    = "deployments/dba-webhooks-${filemd5(local.webhooks_zip_path)}.zip"
  source = local.webhooks_zip_path
  etag   = filemd5(local.webhooks_zip_path)
}

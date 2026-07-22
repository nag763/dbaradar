locals {
  zip_path = "../services/dba/target/function.zip"
}


# 1. Create an S3 Bucket for Lambda artifacts
resource "aws_s3_bucket" "lambda_bucket" {
  bucket        = "dbaradar"
  force_destroy = true
}

resource "aws_s3_object" "lambda_code" {
  bucket = aws_s3_bucket.lambda_bucket.bucket
  key    = "deployments/dba-${filemd5(local.zip_path)}.zip"
  source = local.zip_path

  # Ensures re-upload when local file changes
  etag = filemd5(local.zip_path)
}


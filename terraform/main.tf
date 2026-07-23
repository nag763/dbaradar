terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

variable "region" {
  default = "eu-central-1"
}

# Configure the AWS Provider
provider "aws" {
  region = var.region
}

data "aws_caller_identity" "current" {}
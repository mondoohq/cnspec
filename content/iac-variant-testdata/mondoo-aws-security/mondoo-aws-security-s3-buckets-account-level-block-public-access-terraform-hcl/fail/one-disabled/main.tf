
provider "aws" {
  region = "us-east-1"
}

resource "aws_s3_account_public_access_block" "example" {
  block_public_acls       = true
  ignore_public_acls      = true
  block_public_policy     = true
  restrict_public_buckets = false
}

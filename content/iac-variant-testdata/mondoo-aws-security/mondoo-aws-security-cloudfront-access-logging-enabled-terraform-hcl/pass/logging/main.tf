# Compliant: access logging is configured.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  logging_config {
    bucket = "logs.s3.amazonaws.com"
    prefix = "cf/"
  }
}

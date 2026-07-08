# Non-compliant: counted distributions with no logging_config.
resource "aws_cloudfront_distribution" "edge" {
  count   = 2
  enabled = true
}

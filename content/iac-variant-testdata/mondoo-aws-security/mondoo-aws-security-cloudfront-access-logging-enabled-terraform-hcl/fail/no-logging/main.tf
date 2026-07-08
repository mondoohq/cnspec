# Non-compliant: no logging_config block.
resource "aws_cloudfront_distribution" "example" {
  enabled = true
}

# Two distributions; the second has no logging_config, so .all() must fail.
resource "aws_cloudfront_distribution" "logged" {
  enabled = true
  logging_config {
    bucket = "logs.s3.amazonaws.com"
    prefix = "cf/"
  }
}

resource "aws_cloudfront_distribution" "unlogged" {
  enabled = true
}

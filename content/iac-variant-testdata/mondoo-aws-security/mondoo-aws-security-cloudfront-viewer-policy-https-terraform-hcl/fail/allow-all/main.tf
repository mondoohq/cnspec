# Non-compliant: default cache behavior allows plain HTTP.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  default_cache_behavior {
    target_origin_id       = "s3-origin"
    viewer_protocol_policy = "allow-all"
  }
}

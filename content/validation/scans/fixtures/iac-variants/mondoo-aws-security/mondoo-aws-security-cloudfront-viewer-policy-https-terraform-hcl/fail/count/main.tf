# Non-compliant: counted distributions allow plain HTTP to viewers.
resource "aws_cloudfront_distribution" "edge" {
  count   = 2
  enabled = true
  default_cache_behavior {
    viewer_protocol_policy = "allow-all"
    target_origin_id       = "origin"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
  }
}

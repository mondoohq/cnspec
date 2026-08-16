# Compliant: default cache behavior redirects HTTP to HTTPS.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  default_cache_behavior {
    target_origin_id       = "s3-origin"
    viewer_protocol_policy = "redirect-to-https"
  }
}

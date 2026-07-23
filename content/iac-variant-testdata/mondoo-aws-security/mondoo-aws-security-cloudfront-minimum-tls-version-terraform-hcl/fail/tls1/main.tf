# Non-compliant: minimum protocol version is the outdated TLSv1.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = "TLSv1"
    ssl_support_method       = "sni-only"
  }
}

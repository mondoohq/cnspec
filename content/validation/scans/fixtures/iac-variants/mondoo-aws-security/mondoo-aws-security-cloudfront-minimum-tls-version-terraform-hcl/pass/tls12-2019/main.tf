# Compliant: minimum protocol version is a TLS 1.2 security policy.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = "TLSv1.2_2019"
    ssl_support_method       = "sni-only"
  }
}

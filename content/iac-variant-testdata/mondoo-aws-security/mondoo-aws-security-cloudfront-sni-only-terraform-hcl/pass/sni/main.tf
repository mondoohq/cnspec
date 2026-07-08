# Compliant: viewer certificate uses SNI-only.
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  viewer_certificate {
    acm_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    ssl_support_method  = "sni-only"
  }
}

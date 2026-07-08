# Non-compliant: viewer certificate uses dedicated IP (vip).
resource "aws_cloudfront_distribution" "example" {
  enabled = true

  viewer_certificate {
    acm_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    ssl_support_method  = "vip"
  }
}

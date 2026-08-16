# Non-compliant: counted distributions serve over dedicated static IPs.
resource "aws_cloudfront_distribution" "edge" {
  count   = 2
  enabled = true
  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = "vip"
  }
}

# Two distributions; the second uses dedicated IP (vip) SSL, so .all() must fail.
resource "aws_cloudfront_distribution" "sni" {
  enabled = true
  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = "sni-only"
  }
}

resource "aws_cloudfront_distribution" "vip" {
  enabled = true
  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/def"
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = "vip"
  }
}

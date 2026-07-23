# Non-compliant: a conditional selects dedicated-IP (vip) SSL support when the
# legacy-clients flag is set (it is, by default here).
variable "support_legacy_clients" {
  type    = bool
  default = true
}

resource "aws_cloudfront_distribution" "example" {
  enabled = true
  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = "TLSv1.2_2021"
    ssl_support_method       = var.support_legacy_clients ? "vip" : "sni-only"
  }
}

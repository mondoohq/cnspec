# Non-compliant: a conditional pins the minimum protocol version to the outdated
# TLSv1 when the legacy-clients flag is set (it is, by default here).
variable "support_legacy_clients" {
  type    = bool
  default = true
}

resource "aws_cloudfront_distribution" "example" {
  enabled = true
  viewer_certificate {
    acm_certificate_arn      = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    minimum_protocol_version = var.support_legacy_clients ? "TLSv1" : "TLSv1.2_2021"
    ssl_support_method       = "sni-only"
  }
}

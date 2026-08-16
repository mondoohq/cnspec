# Non-compliant: a conditional allows plain HTTP when the legacy-http flag is set
# (it is, by default here).
variable "allow_legacy_http" {
  type    = bool
  default = true
}

resource "aws_cloudfront_distribution" "example" {
  enabled = true
  default_cache_behavior {
    viewer_protocol_policy = var.allow_legacy_http ? "allow-all" : "redirect-to-https"
    target_origin_id       = "origin"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
  }
}

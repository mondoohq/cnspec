# Non-compliant: distribution has no WAF web ACL associated.
resource "aws_cloudfront_distribution" "fail_example" {
  enabled             = true
  default_root_object = "index.html"
}

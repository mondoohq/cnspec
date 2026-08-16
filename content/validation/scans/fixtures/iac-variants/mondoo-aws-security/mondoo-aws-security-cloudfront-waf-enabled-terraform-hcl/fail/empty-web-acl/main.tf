# Non-compliant: web_acl_id is present but set to an empty string, so no WAF
# web ACL is actually associated with the distribution.
resource "aws_cloudfront_distribution" "fail_example" {
  enabled             = true
  default_root_object = "index.html"
  web_acl_id          = ""
}

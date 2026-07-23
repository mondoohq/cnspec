# Compliant: distribution associates a WAF web ACL.
resource "aws_cloudfront_distribution" "pass_example" {
  enabled     = true
  web_acl_id  = "arn:aws:wafv2:us-east-1:123456789012:global/webacl/example/abc"
  default_root_object = "index.html"
}

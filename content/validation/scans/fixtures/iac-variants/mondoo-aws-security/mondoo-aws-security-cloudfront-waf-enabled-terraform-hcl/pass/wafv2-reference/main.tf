# Compliant: distribution associates a WAFv2 web ACL via a resource reference.
resource "aws_wafv2_web_acl" "example" {
  name  = "example-acl"
  scope = "CLOUDFRONT"

  default_action {
    allow {}
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "example-acl"
    sampled_requests_enabled   = true
  }
}

resource "aws_cloudfront_distribution" "pass_example" {
  enabled             = true
  default_root_object = "index.html"
  web_acl_id          = aws_wafv2_web_acl.example.arn
}

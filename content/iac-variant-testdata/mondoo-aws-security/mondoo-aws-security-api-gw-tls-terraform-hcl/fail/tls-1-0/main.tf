# Non-compliant: custom domain allows the outdated TLS 1.0 security policy.
resource "aws_api_gateway_domain_name" "fail_example" {
  domain_name     = "api.example.com"
  certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
  security_policy = "TLS_1_0"
}

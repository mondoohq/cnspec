# Compliant: custom domain enforces a TLS 1.2 minimum security policy.
resource "aws_api_gateway_domain_name" "pass_example" {
  domain_name     = "api.example.com"
  certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
  security_policy = "TLS_1_2"
}

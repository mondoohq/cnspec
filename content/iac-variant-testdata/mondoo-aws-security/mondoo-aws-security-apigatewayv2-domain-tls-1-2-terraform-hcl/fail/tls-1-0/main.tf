# Non-compliant: v2 domain configuration allows the outdated TLS 1.0.
resource "aws_apigatewayv2_domain_name" "fail_example" {
  domain_name = "api.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_0"
  }
}

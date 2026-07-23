# Non-compliant: a counted domain name allows the outdated TLS 1.0.
resource "aws_apigatewayv2_domain_name" "fail_count" {
  count       = 2
  domain_name = "api-${count.index}.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_0"
  }
}

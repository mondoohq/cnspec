# Non-compliant: one of two domain names allows the outdated TLS 1.0.
resource "aws_apigatewayv2_domain_name" "ok" {
  domain_name = "secure.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

resource "aws_apigatewayv2_domain_name" "bad" {
  domain_name = "legacy.example.com"

  domain_name_configuration {
    certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_0"
  }
}

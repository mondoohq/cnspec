# Compliant: domain configuration enforces a TLS 1.2 minimum security policy.
resource "aws_iot_domain_configuration" "pass_example" {
  name        = "example"
  domain_name = "iot.example.com"

  tls_config {
    security_policy = "IoTSecurityPolicy_TLS_1_2_2022_10"
  }
}

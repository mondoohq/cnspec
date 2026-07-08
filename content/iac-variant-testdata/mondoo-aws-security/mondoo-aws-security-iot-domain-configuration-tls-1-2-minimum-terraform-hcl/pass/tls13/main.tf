# Compliant: domain configuration enforces a TLS 1.3 security policy.
resource "aws_iot_domain_configuration" "pass_example" {
  name        = "example"
  domain_name = "iot.example.com"

  tls_config {
    security_policy = "IoTSecurityPolicy_TLS13_1_2_2022_10"
  }
}

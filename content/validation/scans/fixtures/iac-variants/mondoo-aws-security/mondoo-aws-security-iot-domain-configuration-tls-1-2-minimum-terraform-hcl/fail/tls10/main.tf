# Non-compliant: domain configuration allows a TLS 1.0 security policy.
resource "aws_iot_domain_configuration" "fail_example" {
  name        = "example"
  domain_name = "iot.example.com"

  tls_config {
    security_policy = "IoTSecurityPolicy_TLS_1_0_2015_01"
  }
}

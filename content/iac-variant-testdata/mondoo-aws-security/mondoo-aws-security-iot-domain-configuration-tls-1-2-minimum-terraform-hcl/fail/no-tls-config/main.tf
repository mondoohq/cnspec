# Non-compliant: no tls_config block, so no minimum TLS version is enforced.
resource "aws_iot_domain_configuration" "fail_example" {
  name        = "example"
  domain_name = "iot.example.com"
}

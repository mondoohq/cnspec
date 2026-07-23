# Compliant: certificate uses a strong key algorithm, not RSA_1024.
resource "aws_acm_certificate" "pass_example" {
  domain_name       = "example.com"
  validation_method = "DNS"
  key_algorithm     = "RSA_2048"
}

# Non-compliant: certificate uses the weak RSA_1024 key algorithm.
resource "aws_acm_certificate" "fail_example" {
  domain_name       = "example.com"
  validation_method = "DNS"
  key_algorithm     = "RSA_1024"
}

# Compliant: CA uses a strong SHA384 signing algorithm.
resource "aws_acmpca_certificate_authority" "pass_example" {
  certificate_authority_configuration {
    key_algorithm     = "RSA_2048"
    signing_algorithm = "SHA384WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

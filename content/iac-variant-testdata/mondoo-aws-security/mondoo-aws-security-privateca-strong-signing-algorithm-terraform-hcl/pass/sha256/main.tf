# Compliant: CA uses a strong SHA256 signing algorithm.
resource "aws_acmpca_certificate_authority" "pass_example" {
  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA256WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

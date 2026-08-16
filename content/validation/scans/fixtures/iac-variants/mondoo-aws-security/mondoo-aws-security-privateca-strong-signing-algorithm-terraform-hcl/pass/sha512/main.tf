# Compliant: CA uses a strong SHA512 signing algorithm with ECDSA.
resource "aws_acmpca_certificate_authority" "pass_example" {
  certificate_authority_configuration {
    key_algorithm     = "EC_secp384r1"
    signing_algorithm = "SHA512WITHECDSA"

    subject {
      common_name = "example.com"
    }
  }
}

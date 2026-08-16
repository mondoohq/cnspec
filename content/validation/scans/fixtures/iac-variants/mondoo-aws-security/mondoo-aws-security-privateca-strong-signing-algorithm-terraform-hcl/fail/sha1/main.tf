# Non-compliant: CA uses the weak SHA1 signing algorithm.
resource "aws_acmpca_certificate_authority" "fail_example" {
  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA1WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

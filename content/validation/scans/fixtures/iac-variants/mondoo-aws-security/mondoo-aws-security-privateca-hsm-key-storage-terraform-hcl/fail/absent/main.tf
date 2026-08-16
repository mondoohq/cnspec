# Non-compliant: key_storage_security_standard is not set, so Level 3 HSM storage is not guaranteed.
resource "aws_acmpca_certificate_authority" "fail_absent" {
  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA512WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

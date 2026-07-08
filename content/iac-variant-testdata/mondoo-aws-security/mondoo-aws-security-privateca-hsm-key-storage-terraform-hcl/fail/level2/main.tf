# Non-compliant: CA keys use only FIPS 140-2 Level 2 storage.
resource "aws_acmpca_certificate_authority" "fail_example" {
  key_storage_security_standard = "FIPS_140_2_LEVEL_2_OR_HIGHER"

  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA512WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

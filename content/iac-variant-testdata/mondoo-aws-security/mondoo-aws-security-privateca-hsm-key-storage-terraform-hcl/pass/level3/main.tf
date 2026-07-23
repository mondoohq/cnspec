# Compliant: CA keys are stored in a FIPS 140-2 Level 3 HSM.
resource "aws_acmpca_certificate_authority" "pass_example" {
  key_storage_security_standard = "FIPS_140_2_LEVEL_3_OR_HIGHER"

  certificate_authority_configuration {
    key_algorithm     = "RSA_4096"
    signing_algorithm = "SHA512WITHRSA"

    subject {
      common_name = "example.com"
    }
  }
}

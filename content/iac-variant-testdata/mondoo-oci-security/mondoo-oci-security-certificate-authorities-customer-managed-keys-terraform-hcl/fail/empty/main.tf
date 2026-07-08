# Non-compliant: kms_key_id is present but empty.
resource "oci_certificates_management_certificate_authority" "example" {
  compartment_id = var.compartment_id
  name           = "example-root-ca"
  kms_key_id     = ""

  certificate_authority_config {
    config_type = "ROOT_CA_GENERATED_INTERNALLY"
    subject {
      common_name = "Example Root CA"
    }
    signing_algorithm = "SHA256_WITH_RSA"
  }
}

# Compliant: internally-issued certificate using a strong ECDSA key algorithm.
resource "oci_certificates_management_certificate" "example" {
  compartment_id = var.compartment_id
  name           = "api-tls"

  certificate_config {
    config_type                     = "MANAGED_EXTERNALLY_ISSUED_BY_INTERNAL_CA"
    issuer_certificate_authority_id = oci_certificates_management_certificate_authority.example.id
    key_algorithm                   = "ECDSA_P384"
    signature_algorithm             = "SHA384_WITH_ECDSA"

    subject {
      common_name = "api.example.com"
    }
  }
}

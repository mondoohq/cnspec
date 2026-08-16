# Compliant: internally-issued certificate using a strong RSA key algorithm.
resource "oci_certificates_management_certificate" "example" {
  compartment_id = var.compartment_id
  name           = "web-tls"

  certificate_config {
    config_type                     = "ISSUED_BY_INTERNAL_CA"
    issuer_certificate_authority_id = oci_certificates_management_certificate_authority.example.id
    key_algorithm                   = "RSA2048"
    signature_algorithm             = "SHA256_WITH_RSA"

    subject {
      common_name = "www.example.com"
    }
  }
}

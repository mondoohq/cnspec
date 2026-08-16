# Non-compliant: an internally-issued certificate with no renewal rule at all.
resource "oci_certificates_management_certificate" "example" {
  compartment_id = var.compartment_id
  name           = "web-tls"

  certificate_config {
    config_type                     = "ISSUED_BY_INTERNAL_CA"
    issuer_certificate_authority_id = oci_certificates_management_certificate_authority.example.id
    certificate_profile_type        = "TLS_SERVER"

    subject {
      common_name = "www.example.com"
    }

    validity {
      time_of_validity_not_after = "2035-01-01T00:00:00Z"
    }
  }
}

# Compliant: a certificate issued from the customer's own CSR. OCI holds no
# private key for it and rejects a renewal rule on it, so it is out of scope.
resource "oci_certificates_management_certificate" "example" {
  compartment_id = var.compartment_id
  name           = "web-tls-csr"

  certificate_config {
    config_type                     = "MANAGED_EXTERNALLY_ISSUED_BY_INTERNAL_CA"
    issuer_certificate_authority_id = oci_certificates_management_certificate_authority.example.id
    csr_pem                         = file("web.csr")
  }
}

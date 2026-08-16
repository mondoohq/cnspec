# Compliant (out of scope): an imported certificate has no OCI-managed signature
# algorithm, so it is not evaluated by this check.
resource "oci_certificates_management_certificate" "imported" {
  compartment_id = var.compartment_id
  name           = "imported-tls"

  certificate_config {
    config_type     = "IMPORTED"
    cert_chain_pem  = file("${path.module}/chain.pem")
    certificate_pem = file("${path.module}/cert.pem")
    private_key_pem = file("${path.module}/key.pem")
  }
}

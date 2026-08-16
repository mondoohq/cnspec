# The public primary zone is signed, so a validating resolver can detect a forged answer.
resource "oci_dns_zone" "example_com" {
  compartment_id = var.compartment_ocid
  name           = "example.com"
  zone_type      = "PRIMARY"
  scope          = "GLOBAL"
  dnssec_state   = "ENABLED"
}

# A private zone answers only inside attached VCNs, so it is out of scope and unsigned
# here without failing the check.
resource "oci_dns_zone" "internal" {
  compartment_id = var.compartment_ocid
  name           = "internal.example.com"
  zone_type      = "PRIMARY"
  scope          = "PRIVATE"
  view_id        = oci_dns_view.internal.id
  dnssec_state   = "DISABLED"
}

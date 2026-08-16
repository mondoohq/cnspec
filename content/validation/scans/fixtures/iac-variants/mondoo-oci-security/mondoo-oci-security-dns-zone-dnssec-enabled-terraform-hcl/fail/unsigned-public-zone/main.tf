# An unsigned public zone publishes answers no resolver can verify, so a poisoned cache
# or an on-path substitution is undetectable.
resource "oci_dns_zone" "example_com" {
  compartment_id = var.compartment_ocid
  name           = "example.com"
  zone_type      = "PRIMARY"
  scope          = "GLOBAL"
  dnssec_state   = "DISABLED"
}

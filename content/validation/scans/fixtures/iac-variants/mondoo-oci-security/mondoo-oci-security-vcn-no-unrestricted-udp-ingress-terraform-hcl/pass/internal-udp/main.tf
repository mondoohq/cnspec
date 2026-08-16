# Compliant: an unrestricted UDP rule (no udp_options) is scoped to an internal
# CIDR, not the internet, so it is not subject to the check.
resource "oci_core_security_list" "internal" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "internal-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "10.0.0.0/16"
  }
}

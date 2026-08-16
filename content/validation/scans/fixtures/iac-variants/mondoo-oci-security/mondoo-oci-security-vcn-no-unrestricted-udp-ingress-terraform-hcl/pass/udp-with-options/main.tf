# Compliant: the internet-facing UDP rule restricts the destination port range
# with a udp_options block instead of exposing all UDP ports.
resource "oci_core_security_list" "dns" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "dns-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
    udp_options {
      min = 53
      max = 53
    }
  }
}

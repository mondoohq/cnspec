# Non-compliant: the internet-facing UDP rule has no udp_options block, so it
# exposes every UDP port (0-65535) to the entire internet.
resource "oci_core_security_list" "open" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "open-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
  }
}

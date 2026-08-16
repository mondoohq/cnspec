# Non-compliant: one rule is correctly restricted, but a second internet-facing
# UDP rule exposes all ports because it omits the udp_options block.
resource "oci_core_security_list" "mixed" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "mixed-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
    udp_options {
      min = 53
      max = 53
    }
  }

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
  }
}

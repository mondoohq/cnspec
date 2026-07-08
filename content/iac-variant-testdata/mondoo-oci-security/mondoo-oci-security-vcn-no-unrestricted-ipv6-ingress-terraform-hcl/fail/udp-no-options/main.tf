# Non-compliant: IPv6 internet UDP ingress with no udp_options opens every UDP port.
resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name = "ingress-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "::/0"
  }
}

# Compliant: IPv6 internet ingress is scoped - UDP has options, TCP is limited to HTTPS.
resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name = "ingress-sl"

  ingress_security_rules {
    protocol = "6"
    source   = "::/0"
    tcp_options {
      min = 443
      max = 443
    }
  }

  ingress_security_rules {
    protocol = "17"
    source   = "::/0"
    udp_options {
      min = 443
      max = 443
    }
  }
}

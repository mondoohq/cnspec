# Non-compliant: a stateless rule from ::/0 is internet-facing like one from 0.0.0.0/0.
resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  ingress_security_rules {
    protocol  = "6"
    source    = "::/0"
    stateless = true
    tcp_options {
      min = 443
      max = 443
    }
  }
}

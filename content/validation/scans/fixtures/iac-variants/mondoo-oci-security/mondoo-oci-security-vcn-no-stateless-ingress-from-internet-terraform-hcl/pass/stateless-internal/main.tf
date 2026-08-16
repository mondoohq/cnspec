# Compliant: the only stateless rule is scoped to an internal CIDR, not the internet.
resource "oci_core_security_list" "internal" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "internal-sl"

  ingress_security_rules {
    protocol  = "6"
    source    = "10.0.0.0/16"
    stateless = true
    tcp_options {
      min = 22
      max = 22
    }
  }
}

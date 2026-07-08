# Compliant: an all-protocols egress rule scoped to an internal CIDR.
resource "oci_core_security_list" "egress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name = "egress-sl"

  egress_security_rules {
    protocol    = "all"
    destination = "10.0.0.0/16"
  }
}

# Compliant: unscoped ICMP is only allowed from an internal CIDR.
resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name = "ingress-sl"

  ingress_security_rules {
    protocol = "1"
    source   = "10.0.0.0/16"
  }
}

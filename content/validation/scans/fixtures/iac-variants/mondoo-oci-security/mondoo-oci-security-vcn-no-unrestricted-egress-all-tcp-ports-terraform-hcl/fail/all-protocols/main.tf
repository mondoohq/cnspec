# Non-compliant: egress for all protocols to the internet covers every TCP port.
resource "oci_core_security_list" "egress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "egress-sl"

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }
}

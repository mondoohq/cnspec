# Non-compliant: unscoped TCP egress to ::/0 reaches every TCP port on the internet.
resource "oci_core_security_list" "egress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "egress-sl"

  egress_security_rules {
    protocol    = "6"
    destination = "::/0"
  }
}

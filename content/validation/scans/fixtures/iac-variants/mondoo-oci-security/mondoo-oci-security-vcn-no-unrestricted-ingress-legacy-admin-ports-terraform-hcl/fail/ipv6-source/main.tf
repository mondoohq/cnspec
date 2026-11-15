# Non-compliant: ::/0 is the internet as much as 0.0.0.0/0 is.
resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  ingress_security_rules {
    protocol = "6"
    source   = "::/0"
    tcp_options {
      min = 445
      max = 445
    }
  }
}

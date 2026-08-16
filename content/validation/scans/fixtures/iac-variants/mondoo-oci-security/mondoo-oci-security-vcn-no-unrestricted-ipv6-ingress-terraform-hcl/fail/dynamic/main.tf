# Non-compliant (dynamic block): all-protocol IPv6 internet ingress generated
# via a dynamic "ingress_security_rules" block.
variable "open_v6" {
  default = ["::/0"]
}

resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  dynamic "ingress_security_rules" {
    for_each = var.open_v6
    content {
      protocol = "all"
      source   = ingress_security_rules.value
    }
  }
}

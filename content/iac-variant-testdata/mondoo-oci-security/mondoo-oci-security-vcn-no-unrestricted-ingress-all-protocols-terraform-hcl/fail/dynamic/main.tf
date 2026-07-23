# Non-compliant (dynamic block): all-protocol internet ingress generated from a
# list of CIDRs via a dynamic "ingress_security_rules" block.
variable "open_cidrs" {
  default = ["0.0.0.0/0"]
}

resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  dynamic "ingress_security_rules" {
    for_each = var.open_cidrs
    content {
      protocol = "all"
      source   = ingress_security_rules.value
    }
  }
}

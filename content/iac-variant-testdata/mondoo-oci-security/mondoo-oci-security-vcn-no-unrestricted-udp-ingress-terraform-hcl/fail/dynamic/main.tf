# Non-compliant (dynamic block): a UDP internet rule with no udp_options block
# (opens every UDP port) generated via a dynamic "ingress_security_rules" block.
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
      protocol = "17"
      source   = ingress_security_rules.value
    }
  }
}

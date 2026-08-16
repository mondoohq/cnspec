# Compliant (dynamic block): the dynamically generated UDP internet rule scopes
# the destination ports with a udp_options block.
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
      udp_options {
        min = 53
        max = 53
      }
    }
  }
}

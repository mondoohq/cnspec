# Compliant (dynamic block): the dynamically generated rules only allow all
# protocols from internal CIDRs, never the public internet.
variable "internal_cidrs" {
  default = ["10.0.0.0/16"]
}

resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  dynamic "ingress_security_rules" {
    for_each = var.internal_cidrs
    content {
      protocol = "all"
      source   = ingress_security_rules.value
    }
  }
}

# Non-compliant: the SSH rule's source resolves through a ternary to the public
# internet.
variable "expose_ssh" {
  default = true
}

resource "oci_core_security_list" "ingress" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "ingress-sl"

  ingress_security_rules {
    protocol = "6"
    source   = var.expose_ssh ? "0.0.0.0/0" : "10.0.0.0/16"
    tcp_options {
      min = 22
      max = 22
    }
  }
}

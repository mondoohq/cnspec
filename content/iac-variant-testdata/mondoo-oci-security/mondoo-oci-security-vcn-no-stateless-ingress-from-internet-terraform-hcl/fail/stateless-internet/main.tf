# Non-compliant: a stateless ingress rule is open to the entire internet.
resource "oci_core_security_list" "web" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "web-sl"

  ingress_security_rules {
    protocol  = "6"
    source    = "0.0.0.0/0"
    stateless = true
    tcp_options {
      min = 443
      max = 443
    }
  }
}

# Non-compliant: two security lists, the second has an unscoped UDP internet
# rule (no udp_options). .all() over resources must still flag it.
resource "oci_core_security_list" "dns" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "dns-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
    udp_options {
      min = 53
      max = 53
    }
  }
}

resource "oci_core_security_list" "open" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  vcn_id         = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
  display_name   = "open-sl"

  ingress_security_rules {
    protocol = "17"
    source   = "0.0.0.0/0"
  }
}

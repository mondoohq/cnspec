# The peering joins a VCN in another tenancy. Once routes are published, hosts there are
# simply on this network, with no gateway and no logging point in between, and their
# security lists are outside this tenancy's control.
resource "oci_core_local_peering_gateway" "partner" {
  compartment_id           = var.compartment_ocid
  vcn_id                   = oci_core_vcn.app.id
  display_name             = "partner-integration"
  is_cross_tenancy_peering = true
}

# Both peerings stay inside the tenancy.
resource "oci_core_local_peering_gateway" "app_to_shared" {
  compartment_id           = var.compartment_ocid
  vcn_id                   = oci_core_vcn.app.id
  display_name             = "app-to-shared"
  is_cross_tenancy_peering = false
}

resource "oci_core_remote_peering_connection" "to_dr_region" {
  compartment_id           = var.compartment_ocid
  drg_id                   = oci_core_drg.main.id
  display_name             = "to-dr-region"
  is_cross_tenancy_peering = false
}

# Non-compliant: no kms_key_id, so the volume uses Oracle-managed encryption.
resource "oci_core_volume" "no_cmek" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  availability_domain = "Uocm:PHX-AD-1"
  display_name        = "data-volume"
  size_in_gbs         = 50
}

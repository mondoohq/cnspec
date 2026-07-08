# Non-compliant: no kms_key_id, so Oracle-managed encryption is used.
resource "oci_file_storage_file_system" "no_cmek" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "shared-dev"
}

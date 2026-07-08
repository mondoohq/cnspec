# Non-compliant: kms_key_id set to an empty string.
resource "oci_file_storage_file_system" "empty_cmek" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "shared-test"
  kms_key_id          = ""
}

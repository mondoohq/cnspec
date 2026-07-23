# Non-compliant: kms_key_id is set to an empty string.
resource "oci_core_volume" "empty_cmek" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  availability_domain = "Uocm:PHX-AD-1"
  display_name        = "data-volume"
  size_in_gbs         = 50
  kms_key_id          = ""
}

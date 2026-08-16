# Non-compliant: no auto_key_rotation_details block and rotation left disabled.
resource "oci_kms_key" "no_rotation" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "legacy-key"
  management_endpoint = "https://examplevault-management.kms.us-phoenix-1.oraclecloud.com"

  key_shape {
    algorithm = "AES"
    length    = 32
  }
}

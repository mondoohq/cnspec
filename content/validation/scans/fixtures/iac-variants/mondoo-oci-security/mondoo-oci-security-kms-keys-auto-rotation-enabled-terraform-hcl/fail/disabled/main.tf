# Non-compliant: rotation schedule present but auto rotation explicitly disabled.
resource "oci_kms_key" "disabled_rotation" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "manual-key"
  management_endpoint = "https://examplevault-management.kms.us-phoenix-1.oraclecloud.com"

  key_shape {
    algorithm = "AES"
    length    = 32
  }

  is_auto_rotation_enabled = false

  auto_key_rotation_details {
    rotation_interval_in_days = 90
  }
}

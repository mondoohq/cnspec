# Compliant: automatic key rotation enabled with a rotation schedule.
resource "oci_kms_key" "compliant" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "prod-data-key"
  management_endpoint = "https://examplevault-management.kms.us-phoenix-1.oraclecloud.com"

  key_shape {
    algorithm = "AES"
    length    = 32
  }

  is_auto_rotation_enabled = true

  auto_key_rotation_details {
    rotation_interval_in_days = 90
  }
}

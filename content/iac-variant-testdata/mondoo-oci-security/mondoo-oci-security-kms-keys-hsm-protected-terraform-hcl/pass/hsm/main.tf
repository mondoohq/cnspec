# Compliant: key material is protected by an HSM.
resource "oci_kms_key" "prod" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "prod-data-key"
  management_endpoint = "https://examplevault-management.kms.us-phoenix-1.oraclecloud.com"
  protection_mode     = "HSM"

  key_shape {
    algorithm = "AES"
    length    = 32
  }
}

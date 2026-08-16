# Non-compliant: secret omits rotation_config, so scheduled rotation is not set.
resource "oci_vault_secret" "db_password" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  secret_name    = "db-password"
  vault_id       = "ocid1.vault.oc1.iad.aaaaaaaaexamplevault"
  key_id         = "ocid1.key.oc1.iad.aaaaaaaaexamplekey"

  secret_content {
    content_type = "BASE64"
    content      = "QkVzdHIwMG5nXyMxMQ=="
  }
}

# Compliant: secret declares scheduled rotation and enables it.
resource "oci_vault_secret" "db_password" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  secret_name    = "db-password"
  vault_id       = "ocid1.vault.oc1.iad.aaaaaaaaexamplevault"
  key_id         = "ocid1.key.oc1.iad.aaaaaaaaexamplekey"

  secret_content {
    content_type = "BASE64"
    content      = "QkVzdHIwMG5nXyMxMQ=="
  }

  rotation_config {
    is_scheduled_rotation_enabled = true
    rotation_interval             = "P30D"
    target_system_details {
      target_system_type = "FUNCTION"
      function_id        = "ocid1.fnfunc.oc1.iad.aaaaaaaaexamplefn"
    }
  }
}

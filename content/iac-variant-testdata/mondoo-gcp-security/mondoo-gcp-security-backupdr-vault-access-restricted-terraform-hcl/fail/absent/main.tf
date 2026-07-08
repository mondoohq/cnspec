# Non-compliant: access_restriction is not set.
resource "google_backup_dr_backup_vault" "unset" {
  location                                     = "us-central1"
  backup_vault_id                              = "default-vault"
  backup_minimum_enforced_retention_duration   = "100000s"
}

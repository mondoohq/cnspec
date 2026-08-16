# Non-compliant: backup vault access is unrestricted.
resource "google_backup_dr_backup_vault" "unrestricted" {
  location                                     = "us-central1"
  backup_vault_id                              = "open-vault"
  backup_minimum_enforced_retention_duration   = "100000s"
  access_restriction                           = "UNRESTRICTED"
}

# Compliant: backup vault access restricted to within the project.
resource "google_backup_dr_backup_vault" "within_project" {
  location                                     = "us-central1"
  backup_vault_id                              = "my-vault"
  backup_minimum_enforced_retention_duration   = "100000s"
  access_restriction                           = "WITHIN_PROJECT"
}

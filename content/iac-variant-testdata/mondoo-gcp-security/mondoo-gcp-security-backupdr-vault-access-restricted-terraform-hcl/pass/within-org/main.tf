# Compliant: backup vault access restricted to within the organization.
resource "google_backup_dr_backup_vault" "within_org" {
  location                                     = "us-central1"
  backup_vault_id                              = "org-vault"
  backup_minimum_enforced_retention_duration   = "100000s"
  access_restriction                           = "WITHIN_ORGANIZATION"
}

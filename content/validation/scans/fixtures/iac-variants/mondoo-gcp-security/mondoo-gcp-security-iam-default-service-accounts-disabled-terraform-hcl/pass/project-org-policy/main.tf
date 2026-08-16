# Compliant: legacy organization policy enforces the constraint that disables
# automatic IAM grants for default service accounts.
resource "google_project_organization_policy" "default_sa_grants" {
  project    = "my-project"
  constraint = "iam.automaticIamGrantsForDefaultServiceAccounts"

  boolean_policy {
    enforced = true
  }
}

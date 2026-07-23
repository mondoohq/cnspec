# Non-compliant: legacy organization policy present but not enforced.
resource "google_project_organization_policy" "default_sa_grants" {
  project    = "my-project"
  constraint = "iam.automaticIamGrantsForDefaultServiceAccounts"

  boolean_policy {
    enforced = false
  }
}

# Non-compliant: newer org policy resource present but rule not enforced.
resource "google_org_policy_policy" "default_sa_grants" {
  name   = "projects/my-project/policies/iam.automaticIamGrantsForDefaultServiceAccounts"
  parent = "projects/my-project"

  spec {
    rules {
      enforce = "FALSE"
    }
  }
}

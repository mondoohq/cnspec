# Non-compliant: iam_member grants Token Creator, allowing impersonation.
resource "google_service_account_iam_member" "fail_example" {
  service_account_id = "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "user:attacker@example.com"
}

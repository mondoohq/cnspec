# Compliant: iam_member grants a non-impersonation role.
resource "google_service_account_iam_member" "pass_example" {
  service_account_id = "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountViewer"
  member             = "user:jane@example.com"
}

# Compliant: iam_binding grants a non-impersonation role.
resource "google_service_account_iam_binding" "pass_example" {
  service_account_id = "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountViewer"
  members = [
    "user:jane@example.com",
    "group:admins@example.com",
  ]
}

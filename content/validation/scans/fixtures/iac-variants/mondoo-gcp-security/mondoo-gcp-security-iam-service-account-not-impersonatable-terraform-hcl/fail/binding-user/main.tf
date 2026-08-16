# Non-compliant: iam_binding grants Service Account User, allowing impersonation.
resource "google_service_account_iam_binding" "fail_example" {
  service_account_id = "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  members = [
    "user:attacker@example.com",
  ]
}

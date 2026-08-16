# Non-compliant: IAM member grants access to allUsers (public).
resource "google_secret_manager_secret_iam_member" "fail_example" {
  project   = "my-project"
  secret_id = "my-secret"
  role      = "roles/secretmanager.secretAccessor"
  member    = "allUsers"
}

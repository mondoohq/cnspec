# Non-compliant: binding grants access to allUsers (public).
resource "google_secret_manager_secret_iam_binding" "fail_example" {
  project   = "my-project"
  secret_id = "my-secret"
  role      = "roles/secretmanager.secretAccessor"

  members = [
    "allUsers",
  ]
}

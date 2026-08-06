# Non-compliant: the authoritative binding grants the primitive Owner role.
resource "google_project_iam_binding" "fail_owner" {
  project = "my-project"
  role    = "roles/owner"

  members = [
    "group:admins@example.com",
  ]
}

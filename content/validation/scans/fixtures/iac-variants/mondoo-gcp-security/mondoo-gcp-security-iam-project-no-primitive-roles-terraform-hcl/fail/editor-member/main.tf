# Non-compliant: the member binding grants the primitive Editor role.
resource "google_project_iam_member" "fail_editor" {
  project = "my-project"
  role    = "roles/editor"
  member  = "serviceAccount:build@my-project.iam.gserviceaccount.com"
}

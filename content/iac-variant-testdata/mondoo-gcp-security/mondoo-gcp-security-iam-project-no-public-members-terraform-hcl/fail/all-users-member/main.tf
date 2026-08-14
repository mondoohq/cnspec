# allUsers grants the role to the entire internet, unauthenticated.
resource "google_project_iam_member" "public_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "allUsers"
}

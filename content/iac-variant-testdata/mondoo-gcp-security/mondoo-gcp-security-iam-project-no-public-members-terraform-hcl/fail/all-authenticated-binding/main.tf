# allAuthenticatedUsers is every Google account in existence, not just this
# organization's.
resource "google_project_iam_binding" "storage_readers" {
  project = var.project_id
  role    = "roles/storage.objectViewer"

  members = [
    "group:data-analysts@example.com",
    "allAuthenticatedUsers",
  ]
}

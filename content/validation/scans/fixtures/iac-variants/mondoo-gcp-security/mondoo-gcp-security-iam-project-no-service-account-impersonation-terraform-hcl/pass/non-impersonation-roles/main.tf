resource "google_project_iam_member" "log_viewer" {
  project = var.project_id
  role    = "roles/logging.viewer"
  member  = "group:sre@example.com"
}

resource "google_project_iam_binding" "storage_readers" {
  project = var.project_id
  role    = "roles/storage.objectViewer"

  members = [
    "group:data-analysts@example.com",
  ]
}

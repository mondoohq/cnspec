resource "google_project_iam_member" "log_viewer" {
  project = var.project_id
  role    = "roles/logging.viewer"
  member  = "serviceAccount:log-reader@example-prod.iam.gserviceaccount.com"
}

resource "google_project_iam_binding" "storage_readers" {
  project = var.project_id
  role    = "roles/storage.objectViewer"

  members = [
    "group:data-analysts@example.com",
    "serviceAccount:etl@example-prod.iam.gserviceaccount.com",
  ]
}

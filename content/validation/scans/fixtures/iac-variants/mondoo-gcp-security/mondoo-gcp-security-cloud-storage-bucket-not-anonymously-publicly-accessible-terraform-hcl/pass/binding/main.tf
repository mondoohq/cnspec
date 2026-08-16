# Compliant: binding grants access only to named principals.
resource "google_storage_bucket_iam_binding" "binding" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  members = [
    "group:analytics@example.com",
    "serviceAccount:reporting@my-project.iam.gserviceaccount.com",
  ]
}

# Non-compliant: binding grants public read to allUsers.
resource "google_storage_bucket_iam_binding" "binding" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  members = [
    "group:analytics@example.com",
    "allUsers",
  ]
}

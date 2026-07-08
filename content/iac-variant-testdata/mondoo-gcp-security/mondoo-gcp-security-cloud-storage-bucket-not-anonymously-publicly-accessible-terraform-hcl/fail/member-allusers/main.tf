# Non-compliant: member grants public read to allUsers.
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

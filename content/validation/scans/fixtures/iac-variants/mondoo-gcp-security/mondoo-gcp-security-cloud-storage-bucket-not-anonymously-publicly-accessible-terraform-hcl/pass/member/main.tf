# Compliant: member grants access to a single named user.
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  member = "user:jane@example.com"
}

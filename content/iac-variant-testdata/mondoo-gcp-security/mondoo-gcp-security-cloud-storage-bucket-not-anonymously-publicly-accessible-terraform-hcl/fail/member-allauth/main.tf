# Non-compliant: member grants access to all authenticated users.
resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  member = "allAuthenticatedUsers"
}

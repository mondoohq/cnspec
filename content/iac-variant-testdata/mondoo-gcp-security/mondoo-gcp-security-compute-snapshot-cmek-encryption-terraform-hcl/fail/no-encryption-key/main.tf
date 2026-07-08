# Non-compliant: no snapshot_encryption_key block (Google-managed key only).
resource "google_compute_snapshot" "snap" {
  name        = "db-snapshot"
  source_disk = google_compute_disk.data.id
  zone        = "us-central1-a"
}

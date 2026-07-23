# Non-compliant: uses a customer-supplied raw key, not a KMS-managed key.
resource "google_compute_snapshot" "snap" {
  name        = "db-snapshot"
  source_disk = google_compute_disk.data.id
  zone        = "us-central1-a"

  snapshot_encryption_key {
    raw_key = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
  }
}

# Compliant: snapshot encrypted with a customer-managed KMS key.
resource "google_compute_snapshot" "snap" {
  name        = "db-snapshot"
  source_disk = google_compute_disk.data.id
  zone        = "us-central1-a"

  snapshot_encryption_key {
    kms_key_self_link = google_kms_crypto_key.snapshot.id
  }
}

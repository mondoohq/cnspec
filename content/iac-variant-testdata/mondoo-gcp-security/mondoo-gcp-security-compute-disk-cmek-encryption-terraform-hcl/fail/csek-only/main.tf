# Non-compliant: disk_encryption_key uses a customer-supplied raw key, not a KMS key.
resource "google_compute_disk" "example" {
  name = "csek-disk"
  type = "pd-ssd"
  zone = "us-central1-a"
  size = 50

  disk_encryption_key {
    raw_key = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
  }
}

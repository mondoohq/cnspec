# Compliant: disk encrypted with a customer-supplied raw key.
resource "google_compute_disk" "example" {
  name = "csek-disk"
  type = "pd-ssd"
  zone = "us-central1-a"
  size = 50

  disk_encryption_key {
    raw_key = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
  }
}

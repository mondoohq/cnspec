# Non-compliant: disk has no disk_encryption_key block (Google-managed keys).
resource "google_compute_disk" "example" {
  name = "default-disk"
  type = "pd-ssd"
  zone = "us-central1-a"
  size = 50
}

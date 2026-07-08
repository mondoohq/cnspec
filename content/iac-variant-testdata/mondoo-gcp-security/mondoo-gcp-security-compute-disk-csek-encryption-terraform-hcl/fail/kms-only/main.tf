# Non-compliant: disk_encryption_key uses a KMS key, not a customer-supplied key.
resource "google_compute_disk" "example" {
  name = "cmek-disk"
  type = "pd-ssd"
  zone = "us-central1-a"
  size = 50

  disk_encryption_key {
    kms_key_self_link = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/disk-key"
  }
}

# Non-compliant: no kms_key, so backup collections use Google-managed encryption.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
}

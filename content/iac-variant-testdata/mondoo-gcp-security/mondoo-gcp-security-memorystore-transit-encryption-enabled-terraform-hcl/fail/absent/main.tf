# Non-compliant: transit_encryption_mode omitted.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
}

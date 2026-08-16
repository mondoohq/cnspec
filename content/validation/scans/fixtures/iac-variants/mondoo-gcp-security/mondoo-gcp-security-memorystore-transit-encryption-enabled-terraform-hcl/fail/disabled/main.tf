# Non-compliant: in-transit encryption explicitly disabled.
resource "google_memorystore_instance" "cache" {
  instance_id             = "my-instance"
  location                = "us-central1"
  shard_count             = 3
  transit_encryption_mode = "TRANSIT_ENCRYPTION_DISABLED"
}

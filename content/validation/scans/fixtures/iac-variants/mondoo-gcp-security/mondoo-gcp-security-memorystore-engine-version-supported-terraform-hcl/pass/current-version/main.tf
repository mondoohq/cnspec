# Compliant: a current Valkey line is pinned explicitly.
resource "google_memorystore_instance" "cache" {
  instance_id    = "my-instance"
  location       = "us-central1"
  shard_count    = 3
  engine_version = "VALKEY_9_0"
}

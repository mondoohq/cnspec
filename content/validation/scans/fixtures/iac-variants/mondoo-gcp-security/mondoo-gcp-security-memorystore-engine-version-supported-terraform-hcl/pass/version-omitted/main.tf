# Compliant: engine_version is omitted, so Memorystore assigns the current
# default version rather than the oldest line.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
}

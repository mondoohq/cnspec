# Compliant: engine_configs present without a protected-mode entry.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3

  engine_configs = {
    "maxmemory-policy" = "allkeys-lru"
  }
}

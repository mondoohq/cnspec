# Compliant: protected-mode explicitly enabled.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3

  engine_configs = {
    "maxmemory-policy" = "allkeys-lru"
    "protected-mode"   = "yes"
  }
}

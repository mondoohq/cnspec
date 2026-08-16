# Compliant: no engine_configs at all, so protected-mode keeps its safe default.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
}

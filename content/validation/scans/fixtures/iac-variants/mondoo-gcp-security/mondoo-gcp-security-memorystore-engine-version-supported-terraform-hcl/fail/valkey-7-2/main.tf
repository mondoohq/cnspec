# Non-compliant: pinned to Valkey 7.2, the oldest line Memorystore offers.
resource "google_memorystore_instance" "cache" {
  instance_id    = "my-instance"
  location       = "us-central1"
  shard_count    = 3
  engine_version = "VALKEY_7_2"
}

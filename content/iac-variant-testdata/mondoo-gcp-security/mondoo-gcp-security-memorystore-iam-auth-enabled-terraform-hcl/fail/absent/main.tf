# Non-compliant: authorization_mode omitted (defaults to disabled).
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
}

# Non-compliant: authorization disabled.
resource "google_memorystore_instance" "cache" {
  instance_id        = "my-instance"
  location           = "us-central1"
  shard_count        = 3
  authorization_mode = "AUTH_DISABLED"
}

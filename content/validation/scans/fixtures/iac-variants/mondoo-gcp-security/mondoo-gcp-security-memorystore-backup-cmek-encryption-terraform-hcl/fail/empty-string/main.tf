# Non-compliant: kms_key present but empty.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
  kms_key     = ""
}

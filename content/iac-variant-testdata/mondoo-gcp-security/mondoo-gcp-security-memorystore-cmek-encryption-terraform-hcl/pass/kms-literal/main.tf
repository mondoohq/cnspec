# Compliant: CMEK key supplied as a literal resource path.
resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
  kms_key     = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
}

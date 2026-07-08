# Compliant: CMEK key supplied via a resource reference.
resource "google_kms_crypto_key" "memorystore_key" {
  name     = "memorystore-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"
}

resource "google_memorystore_instance" "cache" {
  instance_id = "my-instance"
  location    = "us-central1"
  shard_count = 3
  kms_key     = google_kms_crypto_key.memorystore_key.id
}

# Compliant: key with no explicit purpose (defaults to ENCRYPT_DECRYPT) has rotation.
resource "google_kms_crypto_key" "pass_example" {
  name     = "default-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "2592000s"
}

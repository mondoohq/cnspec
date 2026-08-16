# Violation: key has no rotation_period configured.
resource "google_kms_crypto_key" "fail_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"
  purpose  = "ENCRYPT_DECRYPT"
}

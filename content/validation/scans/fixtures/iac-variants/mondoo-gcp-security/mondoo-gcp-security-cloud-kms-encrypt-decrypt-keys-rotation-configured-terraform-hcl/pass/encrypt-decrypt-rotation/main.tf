# Compliant: ENCRYPT_DECRYPT key has a rotation_period configured.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s"
}

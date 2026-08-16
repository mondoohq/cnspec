# Compliant (vacuously): no key_access_justifications_policy block present.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "7776000s"
}

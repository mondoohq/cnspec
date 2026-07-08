# Compliant (vacuously): destroy_scheduled_duration not set.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "7776000s"
}

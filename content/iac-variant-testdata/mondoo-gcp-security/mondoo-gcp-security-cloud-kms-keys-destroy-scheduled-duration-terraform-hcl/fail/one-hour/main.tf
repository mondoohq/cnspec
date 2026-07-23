# Violation: destroy scheduled duration of 1 hour (3600s) is below the 24h minimum.
resource "google_kms_crypto_key" "fail_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period            = "7776000s"
  destroy_scheduled_duration = "3600s"
}

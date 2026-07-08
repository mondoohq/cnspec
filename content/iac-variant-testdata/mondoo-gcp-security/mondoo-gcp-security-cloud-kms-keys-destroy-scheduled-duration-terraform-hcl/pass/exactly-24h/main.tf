# Compliant: destroy scheduled duration is exactly 24h (86400s).
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period            = "7776000s"
  destroy_scheduled_duration = "86400s"
}

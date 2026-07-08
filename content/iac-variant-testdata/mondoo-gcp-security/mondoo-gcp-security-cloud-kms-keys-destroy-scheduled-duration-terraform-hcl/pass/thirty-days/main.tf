# Compliant: destroy scheduled duration of 30 days (2592000s), well above 24h.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period            = "7776000s"
  destroy_scheduled_duration = "2592000s"
}

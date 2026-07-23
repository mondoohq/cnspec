# Compliant: key rotates every 30 days (2592000s), under the 90-day cap.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "2592000s"
}

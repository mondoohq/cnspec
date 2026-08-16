# Violation: key rotates every 120 days (10368000s), exceeding the 90-day cap.
resource "google_kms_crypto_key" "fail_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "10368000s"
}

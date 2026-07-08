# Violation: 86399s is one second below the 24h (86400s) minimum. This exercises the
# numeric boundary — a naive lexicographic string compare must not round it up to a pass.
resource "google_kms_crypto_key" "fail_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period            = "7776000s"
  destroy_scheduled_duration = "86399s"
}

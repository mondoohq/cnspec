# Violation: allowed_access_reasons includes CUSTOMER_INITIATED_SUPPORT.
resource "google_kms_crypto_key" "fail_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "7776000s"

  key_access_justifications_policy {
    allowed_access_reasons = [
      "CUSTOMER_INITIATED_ACCESS",
      "CUSTOMER_INITIATED_SUPPORT",
    ]
  }
}

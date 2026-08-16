# Compliant: key access justifications policy does not allow
# CUSTOMER_INITIATED_SUPPORT as a reason.
resource "google_kms_crypto_key" "pass_example" {
  name     = "app-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"

  rotation_period = "7776000s"

  key_access_justifications_policy {
    allowed_access_reasons = [
      "CUSTOMER_INITIATED_ACCESS",
      "GOOGLE_INITIATED_SYSTEM_OPERATION",
    ]
  }
}

# Compliant: an ASYMMETRIC_SIGN key cannot be rotated automatically, so it is
# excluded from the rotation requirement (the check only targets ENCRYPT_DECRYPT).
resource "google_kms_crypto_key" "pass_example" {
  name     = "signing-key"
  key_ring = "projects/my-project/locations/us-central1/keyRings/my-ring"
  purpose  = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }
}

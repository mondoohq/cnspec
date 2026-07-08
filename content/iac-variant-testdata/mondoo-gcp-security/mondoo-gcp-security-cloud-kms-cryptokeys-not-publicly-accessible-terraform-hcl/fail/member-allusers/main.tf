# Violation: crypto key IAM member grants access to allUsers.
resource "google_kms_crypto_key_iam_member" "fail_example" {
  crypto_key_id = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  role          = "roles/cloudkms.cryptoKeyDecrypter"
  member        = "allUsers"
}

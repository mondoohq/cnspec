# Violation: crypto key IAM binding includes allAuthenticatedUsers.
resource "google_kms_crypto_key_iam_binding" "fail_example" {
  crypto_key_id = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "allAuthenticatedUsers",
  ]
}

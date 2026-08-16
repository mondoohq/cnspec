# Compliant: crypto key IAM binding lists only named principals.
resource "google_kms_crypto_key_iam_binding" "pass_example" {
  crypto_key_id = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "group:kms-users@example.com",
  ]
}

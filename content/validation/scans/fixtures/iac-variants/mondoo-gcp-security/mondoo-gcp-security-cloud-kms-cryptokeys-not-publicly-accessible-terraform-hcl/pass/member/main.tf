# Compliant: crypto key IAM member is a named service account.
resource "google_kms_crypto_key_iam_member" "pass_example" {
  crypto_key_id = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}

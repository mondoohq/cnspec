# Compliant: key ring IAM member is a named service account.
resource "google_kms_key_ring_iam_member" "pass_example" {
  key_ring_id = "projects/my-project/locations/us-central1/keyRings/my-ring"
  role        = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member      = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}

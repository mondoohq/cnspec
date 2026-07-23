# Violation: key ring IAM binding includes allAuthenticatedUsers.
resource "google_kms_key_ring_iam_binding" "fail_example" {
  key_ring_id = "projects/my-project/locations/us-central1/keyRings/my-ring"
  role        = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "allAuthenticatedUsers",
  ]
}

# Violation: key ring IAM member grants access to allUsers.
resource "google_kms_key_ring_iam_member" "fail_example" {
  key_ring_id = "projects/my-project/locations/us-central1/keyRings/my-ring"
  role        = "roles/cloudkms.cryptoKeyDecrypter"
  member      = "allUsers"
}

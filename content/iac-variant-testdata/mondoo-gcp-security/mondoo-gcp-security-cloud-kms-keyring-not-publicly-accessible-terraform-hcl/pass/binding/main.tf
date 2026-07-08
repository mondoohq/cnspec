# Compliant: key ring IAM binding lists only named principals.
resource "google_kms_key_ring_iam_binding" "pass_example" {
  key_ring_id = "projects/my-project/locations/us-central1/keyRings/my-ring"
  role        = "roles/cloudkms.viewer"
  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "group:kms-admins@example.com",
  ]
}

# Non-compliant: IAM member grants access to allUsers (public).
resource "google_compute_snapshot_iam_member" "fail_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.viewer"
  member   = "allUsers"
}

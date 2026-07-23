# Non-compliant: IAM member grants the primitive roles/owner role.
resource "google_compute_snapshot_iam_member" "fail_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/owner"
  member   = "user:alice@example.com"
}

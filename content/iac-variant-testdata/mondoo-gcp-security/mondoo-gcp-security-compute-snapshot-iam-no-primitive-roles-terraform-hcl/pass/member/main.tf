# Compliant: IAM member grants a predefined role, not a primitive role.
resource "google_compute_snapshot_iam_member" "pass_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.storageAdmin"
  member   = "user:alice@example.com"
}

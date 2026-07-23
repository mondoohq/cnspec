# Compliant: IAM binding grants a predefined role, not a primitive role.
resource "google_compute_snapshot_iam_binding" "pass_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.viewer"

  members = [
    "group:compute-admins@example.com",
  ]
}

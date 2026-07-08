# Non-compliant: IAM binding grants the primitive roles/editor role.
resource "google_compute_snapshot_iam_binding" "fail_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/editor"

  members = [
    "group:developers@example.com",
  ]
}

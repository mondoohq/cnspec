# Non-compliant: IAM binding grants access to allAuthenticatedUsers (public).
resource "google_compute_snapshot_iam_binding" "fail_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.viewer"

  members = [
    "group:analysts@example.com",
    "allAuthenticatedUsers",
  ]
}

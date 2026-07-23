# Non-compliant: instance IAM binding grants access to allAuthenticatedUsers.
resource "google_spanner_instance_iam_binding" "fail_example" {
  instance = "my-instance"
  role     = "roles/spanner.viewer"

  members = [
    "group:db-ops@example.com",
    "allAuthenticatedUsers",
  ]
}

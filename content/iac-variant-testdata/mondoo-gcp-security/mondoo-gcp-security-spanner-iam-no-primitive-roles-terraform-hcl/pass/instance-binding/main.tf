# Compliant: instance IAM binding uses a predefined non-primitive role.
resource "google_spanner_instance_iam_binding" "pass_example" {
  instance = "my-instance"
  role     = "roles/spanner.viewer"

  members = [
    "group:db-ops@example.com",
  ]
}

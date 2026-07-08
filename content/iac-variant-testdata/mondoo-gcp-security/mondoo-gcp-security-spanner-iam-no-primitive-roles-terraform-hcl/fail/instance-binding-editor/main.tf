# Non-compliant: instance IAM binding grants the primitive roles/editor.
resource "google_spanner_instance_iam_binding" "fail_example" {
  instance = "my-instance"
  role     = "roles/editor"

  members = [
    "group:db-ops@example.com",
  ]
}

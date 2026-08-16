# Non-compliant: instance IAM member grants access to allUsers (public).
resource "google_spanner_instance_iam_member" "fail_example" {
  instance = "my-instance"
  role     = "roles/spanner.viewer"
  member   = "allUsers"
}

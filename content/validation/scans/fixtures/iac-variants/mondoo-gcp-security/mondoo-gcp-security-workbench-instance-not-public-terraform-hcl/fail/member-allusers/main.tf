# Non-compliant: member grants access to allUsers.
resource "google_workbench_instance_iam_member" "fail_example" {
  name     = "analysis-instance"
  location = "us-central1-a"
  role   = "roles/viewer"
  member = "allUsers"
}

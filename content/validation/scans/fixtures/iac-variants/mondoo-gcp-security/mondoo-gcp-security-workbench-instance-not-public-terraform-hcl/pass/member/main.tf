# Compliant: access granted to a named group.
resource "google_workbench_instance_iam_member" "pass_example" {
  name     = "analysis-instance"
  location = "us-central1-a"
  role   = "roles/viewer"
  member = "group:team@example.com"
}

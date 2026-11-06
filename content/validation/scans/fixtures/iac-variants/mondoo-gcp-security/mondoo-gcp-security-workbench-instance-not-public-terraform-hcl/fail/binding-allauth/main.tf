# Non-compliant: binding includes allAuthenticatedUsers.
resource "google_workbench_instance_iam_binding" "fail_example" {
  name     = "analysis-instance"
  location = "us-central1-a"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "allAuthenticatedUsers",
  ]
}

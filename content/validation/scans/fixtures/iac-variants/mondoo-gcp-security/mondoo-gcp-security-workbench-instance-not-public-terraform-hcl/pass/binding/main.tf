# Compliant: binding lists only named principals.
resource "google_workbench_instance_iam_binding" "pass_example" {
  name     = "analysis-instance"
  location = "us-central1-a"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

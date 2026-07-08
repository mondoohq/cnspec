# Compliant: binding grants a predefined Cloud Tasks role.
resource "google_cloud_tasks_queue_iam_binding" "binding" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/cloudtasks.viewer"
  members = [
    "group:platform@example.com",
  ]
}

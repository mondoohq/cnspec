# Non-compliant: binding exposes the queue to all authenticated users.
resource "google_cloud_tasks_queue_iam_binding" "binding" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/cloudtasks.enqueuer"
  members = [
    "group:platform@example.com",
    "allAuthenticatedUsers",
  ]
}

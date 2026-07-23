# Non-compliant: binding grants the primitive roles/editor role.
resource "google_cloud_tasks_queue_iam_binding" "binding" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/editor"
  members = [
    "group:developers@example.com",
  ]
}

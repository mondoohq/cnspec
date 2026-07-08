# Non-compliant: member exposes the queue to allUsers.
resource "google_cloud_tasks_queue_iam_member" "member" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/cloudtasks.enqueuer"
  member   = "allUsers"
}

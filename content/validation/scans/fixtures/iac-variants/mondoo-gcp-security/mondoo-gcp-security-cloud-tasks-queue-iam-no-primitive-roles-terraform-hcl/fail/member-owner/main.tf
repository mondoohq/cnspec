# Non-compliant: member grants the primitive roles/owner role.
resource "google_cloud_tasks_queue_iam_member" "member" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/owner"
  member   = "user:alice@example.com"
}

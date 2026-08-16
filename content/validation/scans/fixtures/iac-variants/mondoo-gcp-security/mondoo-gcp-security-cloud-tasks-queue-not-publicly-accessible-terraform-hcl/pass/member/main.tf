# Compliant: member grants access to a named service account.
resource "google_cloud_tasks_queue_iam_member" "member" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/cloudtasks.enqueuer"
  member   = "serviceAccount:worker@my-project.iam.gserviceaccount.com"
}

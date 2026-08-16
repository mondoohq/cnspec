# Compliant: binding grants access only to named principals.
resource "google_cloud_tasks_queue_iam_binding" "binding" {
  name     = "my-queue"
  location = "us-central1"
  role     = "roles/cloudtasks.enqueuer"
  members = [
    "group:platform@example.com",
    "serviceAccount:worker@my-project.iam.gserviceaccount.com",
  ]
}

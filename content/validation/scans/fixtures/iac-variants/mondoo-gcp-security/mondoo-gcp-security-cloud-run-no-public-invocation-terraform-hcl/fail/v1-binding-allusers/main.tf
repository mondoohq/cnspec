# Non-compliant: binding includes allUsers among its members.
resource "google_cloud_run_service_iam_binding" "fail_example" {
  location = "us-central1"
  service  = "web-app"
  role     = "roles/run.invoker"
  members = [
    "user:alice@example.com",
    "allUsers",
  ]
}

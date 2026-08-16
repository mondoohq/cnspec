# Compliant: invoker granted to a specific principal, not a public group.
resource "google_cloud_run_service_iam_member" "pass_example" {
  location = "us-central1"
  service  = "web-app"
  role     = "roles/run.invoker"
  member   = "user:alice@example.com"
}

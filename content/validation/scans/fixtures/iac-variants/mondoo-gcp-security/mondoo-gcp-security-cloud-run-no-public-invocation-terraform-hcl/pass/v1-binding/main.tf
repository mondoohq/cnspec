# Compliant: binding lists only specific principals.
resource "google_cloud_run_service_iam_binding" "pass_example" {
  location = "us-central1"
  service  = "web-app"
  role     = "roles/run.invoker"
  members = [
    "user:alice@example.com",
    "group:sre@example.com",
  ]
}

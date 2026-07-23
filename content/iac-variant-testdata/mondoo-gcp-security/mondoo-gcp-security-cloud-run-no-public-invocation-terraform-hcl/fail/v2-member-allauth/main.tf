# Non-compliant: v2 invoker granted to allAuthenticatedUsers.
resource "google_cloud_run_v2_service_iam_member" "fail_example" {
  location = "us-central1"
  name     = "web-app"
  role     = "roles/run.invoker"
  member   = "allAuthenticatedUsers"
}

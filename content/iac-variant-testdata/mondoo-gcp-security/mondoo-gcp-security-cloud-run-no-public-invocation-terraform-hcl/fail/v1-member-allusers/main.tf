# Non-compliant: invoker granted to allUsers (public, unauthenticated).
resource "google_cloud_run_service_iam_member" "fail_example" {
  location = "us-central1"
  service  = "web-app"
  role     = "roles/run.invoker"
  member   = "allUsers"
}

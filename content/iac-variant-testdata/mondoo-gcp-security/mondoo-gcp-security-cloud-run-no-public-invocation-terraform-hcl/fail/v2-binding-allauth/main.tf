# Non-compliant: v2 binding includes allAuthenticatedUsers among its members.
resource "google_cloud_run_v2_service_iam_binding" "fail_example" {
  location = "us-central1"
  name     = "web-app"
  role     = "roles/run.invoker"
  members = [
    "allAuthenticatedUsers",
  ]
}

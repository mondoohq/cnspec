# Compliant: v2 invoker granted to a service account.
resource "google_cloud_run_v2_service_iam_member" "pass_example" {
  location = "us-central1"
  name     = "web-app"
  role     = "roles/run.invoker"
  member   = "serviceAccount:caller@my-project.iam.gserviceaccount.com"
}

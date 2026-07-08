# Compliant: v2 binding lists only specific principals.
resource "google_cloud_run_v2_service_iam_binding" "pass_example" {
  location = "us-central1"
  name     = "web-app"
  role     = "roles/run.invoker"
  members = [
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

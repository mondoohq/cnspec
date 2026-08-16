# Compliant: gen2 invoker binding lists only named principals.
resource "google_cloudfunctions2_function_iam_binding" "pass_example" {
  project        = "my-project"
  location       = "us-central1"
  cloud_function = "app-fn-v2"
  role           = "roles/run.invoker"
  members = [
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}

# Compliant: gen1 invoker binding lists only named principals.
resource "google_cloudfunctions_function_iam_binding" "pass_example" {
  project        = "my-project"
  region         = "us-central1"
  cloud_function = "app-fn"
  role           = "roles/cloudfunctions.invoker"
  members = [
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
    "user:alice@example.com",
  ]
}

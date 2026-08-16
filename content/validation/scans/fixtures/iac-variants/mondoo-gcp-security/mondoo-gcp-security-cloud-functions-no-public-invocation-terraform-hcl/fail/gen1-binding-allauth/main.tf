# Violation: gen1 invoker binding includes allAuthenticatedUsers.
resource "google_cloudfunctions_function_iam_binding" "fail_example" {
  project        = "my-project"
  region         = "us-central1"
  cloud_function = "app-fn"
  role           = "roles/cloudfunctions.invoker"
  members = [
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
    "allAuthenticatedUsers",
  ]
}

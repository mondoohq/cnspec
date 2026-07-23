# Violation: gen2 invoker binding includes allUsers.
resource "google_cloudfunctions2_function_iam_binding" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  cloud_function = "app-fn-v2"
  role           = "roles/run.invoker"
  members = [
    "allUsers",
  ]
}

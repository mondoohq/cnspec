# Violation: gen2 invoker granted to allUsers (public invocation).
resource "google_cloudfunctions2_function_iam_member" "fail_example" {
  project        = "my-project"
  location       = "us-central1"
  cloud_function = "app-fn-v2"
  role           = "roles/run.invoker"
  member         = "allUsers"
}

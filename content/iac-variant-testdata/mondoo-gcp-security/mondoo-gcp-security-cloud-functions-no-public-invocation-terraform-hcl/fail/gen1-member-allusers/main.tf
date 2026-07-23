# Violation: gen1 invoker granted to allUsers (public invocation).
resource "google_cloudfunctions_function_iam_member" "fail_example" {
  project        = "my-project"
  region         = "us-central1"
  cloud_function = "app-fn"
  role           = "roles/cloudfunctions.invoker"
  member         = "allUsers"
}

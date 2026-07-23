# Compliant: gen2 invoker granted to a named user, not public.
resource "google_cloudfunctions2_function_iam_member" "pass_example" {
  project        = "my-project"
  location       = "us-central1"
  cloud_function = "app-fn-v2"
  role           = "roles/cloudfunctions.invoker"
  member         = "user:alice@example.com"
}

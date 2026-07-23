# Compliant: gen1 invoker granted to a specific service account, not public.
resource "google_cloudfunctions_function_iam_member" "pass_example" {
  project        = "my-project"
  region         = "us-central1"
  cloud_function = "app-fn"
  role           = "roles/cloudfunctions.invoker"
  member         = "serviceAccount:caller@my-project.iam.gserviceaccount.com"
}

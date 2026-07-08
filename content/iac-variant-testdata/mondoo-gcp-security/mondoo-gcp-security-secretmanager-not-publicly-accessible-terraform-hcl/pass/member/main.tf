# Compliant: IAM member is a specific service account, not public.
resource "google_secret_manager_secret_iam_member" "pass_example" {
  project   = "my-project"
  secret_id = "my-secret"
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}

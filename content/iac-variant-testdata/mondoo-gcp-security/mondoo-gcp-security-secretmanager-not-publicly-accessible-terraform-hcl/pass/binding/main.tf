# Compliant: binding members are specific principals, not public.
resource "google_secret_manager_secret_iam_binding" "pass_example" {
  project   = "my-project"
  secret_id = "my-secret"
  role      = "roles/secretmanager.secretAccessor"

  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "group:secrets-readers@example.com",
  ]
}

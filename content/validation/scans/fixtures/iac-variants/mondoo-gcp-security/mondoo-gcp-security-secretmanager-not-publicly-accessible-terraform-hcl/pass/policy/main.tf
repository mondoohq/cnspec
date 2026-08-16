# Compliant: authoritative IAM policy binds a specific service account only.
data "google_iam_policy" "secret" {
  binding {
    role    = "roles/secretmanager.secretAccessor"
    members = ["serviceAccount:app@my-project.iam.gserviceaccount.com"]
  }
}

resource "google_secret_manager_secret_iam_policy" "pass_example" {
  project     = "my-project"
  secret_id   = "my-secret"
  policy_data = data.google_iam_policy.secret.policy_data
}

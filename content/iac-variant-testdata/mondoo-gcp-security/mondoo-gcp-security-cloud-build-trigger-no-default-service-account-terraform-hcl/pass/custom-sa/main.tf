# Compliant: trigger runs as a dedicated, non-default service account.
resource "google_cloudbuild_trigger" "pass_example" {
  name            = "pass-trigger"
  location        = "us-central1"
  service_account = "projects/my-project/serviceAccounts/ci-builder@my-project.iam.gserviceaccount.com"

  github {
    owner = "my-org"
    name  = "my-repo"
    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"
}

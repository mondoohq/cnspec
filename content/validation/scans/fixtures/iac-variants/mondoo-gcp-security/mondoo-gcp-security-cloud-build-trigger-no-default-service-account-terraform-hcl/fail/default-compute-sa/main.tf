# Non-compliant: trigger runs as the default Compute Engine service account.
resource "google_cloudbuild_trigger" "fail_example" {
  name            = "fail-trigger"
  location        = "us-central1"
  service_account = "projects/my-project/serviceAccounts/123456789012-compute@developer.gserviceaccount.com"

  github {
    owner = "my-org"
    name  = "my-repo"
    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"
}

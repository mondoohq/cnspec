# Non-compliant: no service_account set, so the trigger uses the default.
resource "google_cloudbuild_trigger" "fail_example" {
  name     = "fail-trigger"
  location = "us-central1"

  github {
    owner = "my-org"
    name  = "my-repo"
    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"
}

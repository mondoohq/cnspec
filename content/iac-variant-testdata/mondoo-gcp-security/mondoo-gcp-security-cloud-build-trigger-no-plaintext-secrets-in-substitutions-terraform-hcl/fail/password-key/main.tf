# Non-compliant: a substitution key holds a plaintext database password.
resource "google_cloudbuild_trigger" "fail_example" {
  name     = "fail-trigger"
  location = "us-central1"

  substitutions = {
    _DEPLOY_REGION = "us-central1"
    _DB_PASSWORD   = "sup3rs3cret"
  }

  github {
    owner = "my-org"
    name  = "my-repo"
    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"
}

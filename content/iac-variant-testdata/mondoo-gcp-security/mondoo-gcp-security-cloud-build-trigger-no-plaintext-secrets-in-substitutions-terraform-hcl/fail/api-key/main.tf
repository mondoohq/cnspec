# Non-compliant: a substitution key holds a plaintext API key.
resource "google_cloudbuild_trigger" "fail_example" {
  name     = "fail-trigger"
  location = "us-central1"

  substitutions = {
    _DEPLOY_REGION = "us-central1"
    _API_KEY       = "AIzaSyExampleKeyValue1234567890"
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

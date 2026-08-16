# Compliant: substitutions contain only non-sensitive build parameters.
resource "google_cloudbuild_trigger" "pass_example" {
  name     = "pass-trigger"
  location = "us-central1"

  substitutions = {
    _DEPLOY_REGION = "us-central1"
    _SERVICE_NAME  = "my-service"
    _IMAGE_TAG     = "latest"
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

# Compliant: trigger defines no substitutions at all.
resource "google_cloudbuild_trigger" "pass_example" {
  name     = "pass-trigger"
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

# Non-compliant: environment has no config block at all.
resource "google_composer_environment" "example" {
  name   = "bare-environment"
  region = "us-central1"
}

# Non-compliant: no config block at all, no CMEK configured.
resource "google_composer_environment" "bare" {
  name   = "bare-composer"
  region = "us-central1"
}

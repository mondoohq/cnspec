# Non-compliant: config present but no encryption_config (Google-managed keys).
resource "google_composer_environment" "default" {
  name   = "dev-composer"
  region = "us-central1"

  config {
    node_config {
      network    = "default"
      subnetwork = "default"
    }
  }
}

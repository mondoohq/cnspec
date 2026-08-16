# Non-compliant: config block present but no private_environment_config block.
resource "google_composer_environment" "example" {
  name   = "public-environment"
  region = "us-central1"

  config {
    software_config {
      image_version = "composer-2-airflow-2"
    }
  }
}

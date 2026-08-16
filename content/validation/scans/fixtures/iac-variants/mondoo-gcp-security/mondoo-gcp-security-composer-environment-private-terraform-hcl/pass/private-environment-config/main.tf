# Compliant: config block contains a private_environment_config block.
resource "google_composer_environment" "example" {
  name   = "example-environment"
  region = "us-central1"

  config {
    software_config {
      image_version = "composer-2-airflow-2"
    }

    private_environment_config {
      enable_private_endpoint = true
    }
  }
}

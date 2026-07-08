# Compliant: Composer environment encrypted with a customer-managed key.
resource "google_composer_environment" "cmek" {
  name   = "prod-composer"
  region = "us-central1"

  config {
    encryption_config {
      kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/composer-key"
    }
  }
}

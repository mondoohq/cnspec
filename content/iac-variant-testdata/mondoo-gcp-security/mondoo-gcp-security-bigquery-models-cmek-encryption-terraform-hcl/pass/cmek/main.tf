# Compliant: dataset default encryption uses a customer-managed key, so models
# created in it inherit CMEK protection.
resource "google_bigquery_dataset" "cmek" {
  dataset_id = "my_models_dataset"
  location   = "US"

  default_encryption_configuration {
    kms_key_name = "projects/my-project/locations/us/keyRings/my-ring/cryptoKeys/my-key"
  }
}

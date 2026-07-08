# Compliant: dataset encrypted with a customer-managed key (CMEK).
resource "google_bigquery_dataset" "cmek" {
  dataset_id = "my_dataset"
  location   = "US"

  default_encryption_configuration {
    kms_key_name = "projects/my-project/locations/us/keyRings/my-ring/cryptoKeys/my-key"
  }
}

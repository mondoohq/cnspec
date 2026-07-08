# Non-compliant: no default_encryption_configuration, so models use Google-managed keys.
resource "google_bigquery_dataset" "default_keys" {
  dataset_id = "my_models_dataset"
  location   = "US"
}

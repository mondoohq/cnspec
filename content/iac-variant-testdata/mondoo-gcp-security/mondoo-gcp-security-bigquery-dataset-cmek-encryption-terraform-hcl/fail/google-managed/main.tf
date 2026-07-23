# Non-compliant: no default_encryption_configuration, so Google-managed keys are used.
resource "google_bigquery_dataset" "default_keys" {
  dataset_id = "my_dataset"
  location   = "US"
}

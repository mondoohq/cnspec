# Compliant: table has an encryption_configuration block with a KMS key.
resource "google_bigquery_dataset" "example" {
  dataset_id = "example_dataset"
  location   = "US"
}

resource "google_bigquery_table" "pass_example" {
  dataset_id = google_bigquery_dataset.example.dataset_id
  table_id   = "pass_table"

  encryption_configuration {
    kms_key_name = "projects/my-project/locations/us/keyRings/my-ring/cryptoKeys/my-key"
  }

  schema = <<EOF
[
  {"name": "id", "type": "STRING", "mode": "REQUIRED"}
]
EOF
}

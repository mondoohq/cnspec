# Non-compliant: encryption_configuration block present but kms_key_name is empty.
resource "google_bigquery_dataset" "example" {
  dataset_id = "example_dataset"
  location   = "US"
}

resource "google_bigquery_table" "fail_example" {
  dataset_id = google_bigquery_dataset.example.dataset_id
  table_id   = "fail_table"

  encryption_configuration {
    kms_key_name = ""
  }

  schema = <<EOF
[
  {"name": "id", "type": "STRING", "mode": "REQUIRED"}
]
EOF
}

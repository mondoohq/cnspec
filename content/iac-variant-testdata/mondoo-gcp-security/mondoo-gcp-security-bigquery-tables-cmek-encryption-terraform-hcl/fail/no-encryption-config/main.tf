# Non-compliant: table has no encryption_configuration block (Google-managed keys).
resource "google_bigquery_dataset" "example" {
  dataset_id = "example_dataset"
  location   = "US"
}

resource "google_bigquery_table" "fail_example" {
  dataset_id = google_bigquery_dataset.example.dataset_id
  table_id   = "fail_table"

  schema = <<EOF
[
  {"name": "id", "type": "STRING", "mode": "REQUIRED"}
]
EOF
}

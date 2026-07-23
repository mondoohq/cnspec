# Non-compliant: view uses legacy SQL (use_legacy_sql = true).
resource "google_bigquery_dataset" "example" {
  dataset_id = "example_dataset"
  location   = "US"
}

resource "google_bigquery_table" "fail_view" {
  dataset_id = google_bigquery_dataset.example.dataset_id
  table_id   = "fail_view"

  view {
    query          = "SELECT id, name FROM [my-project:example_dataset.source_table]"
    use_legacy_sql = true
  }
}

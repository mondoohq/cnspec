# Compliant: view uses standard SQL (use_legacy_sql = false).
resource "google_bigquery_dataset" "example" {
  dataset_id = "example_dataset"
  location   = "US"
}

resource "google_bigquery_table" "pass_view" {
  dataset_id = google_bigquery_dataset.example.dataset_id
  table_id   = "pass_view"

  view {
    query          = "SELECT id, name FROM `my-project.example_dataset.source_table`"
    use_legacy_sql = false
  }
}

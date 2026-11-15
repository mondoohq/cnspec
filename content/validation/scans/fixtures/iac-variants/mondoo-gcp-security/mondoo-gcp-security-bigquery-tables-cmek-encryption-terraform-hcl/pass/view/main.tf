# Compliant: a view stores no data of its own, so it carries no encryption configuration.
resource "google_bigquery_table" "pass_view" {
  dataset_id = "my_dataset"
  table_id   = "my_view"

  view {
    query          = "SELECT 1 AS a"
    use_legacy_sql = false
  }
}

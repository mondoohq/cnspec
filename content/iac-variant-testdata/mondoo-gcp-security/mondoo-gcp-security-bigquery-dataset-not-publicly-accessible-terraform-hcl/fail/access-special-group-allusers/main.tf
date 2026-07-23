# Non-compliant: dataset access block grants to allUsers via special_group.
resource "google_bigquery_dataset" "public" {
  dataset_id = "my_dataset"
  location   = "US"

  access {
    role          = "READER"
    special_group = "allUsers"
  }
}

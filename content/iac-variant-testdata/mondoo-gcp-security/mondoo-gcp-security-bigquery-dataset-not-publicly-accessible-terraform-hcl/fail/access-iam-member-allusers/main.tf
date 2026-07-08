# Non-compliant: dataset access block grants to allUsers via iam_member.
resource "google_bigquery_dataset" "public" {
  dataset_id = "my_dataset"
  location   = "US"

  access {
    role       = "READER"
    iam_member = "allUsers"
  }
}

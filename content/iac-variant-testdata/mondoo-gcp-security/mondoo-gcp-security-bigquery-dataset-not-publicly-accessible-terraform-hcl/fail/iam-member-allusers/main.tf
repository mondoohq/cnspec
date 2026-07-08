# Non-compliant: dataset IAM member grants access to allUsers.
resource "google_bigquery_dataset" "ds" {
  dataset_id = "my_dataset"
  location   = "US"
}

resource "google_bigquery_dataset_iam_member" "public" {
  dataset_id = google_bigquery_dataset.ds.dataset_id
  role       = "roles/bigquery.dataViewer"
  member     = "allUsers"
}

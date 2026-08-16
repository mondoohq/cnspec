# Compliant: dataset IAM member grants access to a named user.
resource "google_bigquery_dataset" "ds" {
  dataset_id = "my_dataset"
  location   = "US"
}

resource "google_bigquery_dataset_iam_member" "reader" {
  dataset_id = google_bigquery_dataset.ds.dataset_id
  role       = "roles/bigquery.dataViewer"
  member     = "user:analyst@example.com"
}

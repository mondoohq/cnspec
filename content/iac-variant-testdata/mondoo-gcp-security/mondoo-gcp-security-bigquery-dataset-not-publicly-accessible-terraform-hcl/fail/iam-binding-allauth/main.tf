# Non-compliant: dataset IAM binding includes allAuthenticatedUsers.
resource "google_bigquery_dataset" "ds" {
  dataset_id = "my_dataset"
  location   = "US"
}

resource "google_bigquery_dataset_iam_binding" "public" {
  dataset_id = google_bigquery_dataset.ds.dataset_id
  role       = "roles/bigquery.dataViewer"
  members = [
    "group:analysts@example.com",
    "allAuthenticatedUsers",
  ]
}

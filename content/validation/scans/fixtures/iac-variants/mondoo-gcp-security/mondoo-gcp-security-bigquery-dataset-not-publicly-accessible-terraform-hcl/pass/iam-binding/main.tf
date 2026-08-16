# Compliant: dataset IAM binding grants access to named principals only.
resource "google_bigquery_dataset" "ds" {
  dataset_id = "my_dataset"
  location   = "US"
}

resource "google_bigquery_dataset_iam_binding" "readers" {
  dataset_id = google_bigquery_dataset.ds.dataset_id
  role       = "roles/bigquery.dataViewer"
  members = [
    "group:analysts@example.com",
    "serviceAccount:etl@my-project.iam.gserviceaccount.com",
  ]
}

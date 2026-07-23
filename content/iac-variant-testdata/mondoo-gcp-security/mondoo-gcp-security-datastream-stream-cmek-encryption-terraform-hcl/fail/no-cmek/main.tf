# Non-compliant: stream omits customer_managed_encryption_key, using Google-managed keys.
resource "google_datastream_stream" "google_managed" {
  display_name  = "app-stream"
  location      = "us-central1"
  stream_id     = "app-stream"
  desired_state = "RUNNING"

  source_config {
    source_connection_profile = "projects/my-project/locations/us-central1/connectionProfiles/mysql-source"
    mysql_source_config {}
  }

  destination_config {
    destination_connection_profile = "projects/my-project/locations/us-central1/connectionProfiles/bq-dest"
    bigquery_destination_config {
      data_freshness = "900s"
      single_target_dataset {
        dataset_id = "projects/my-project/datasets/app"
      }
    }
  }

  backfill_none {}
}

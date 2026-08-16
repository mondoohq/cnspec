# Compliant: stream is encrypted with a customer-managed encryption key.
resource "google_datastream_stream" "compliant" {
  display_name  = "app-stream"
  location      = "us-central1"
  stream_id     = "app-stream"
  desired_state = "RUNNING"

  customer_managed_encryption_key = "projects/my-project/locations/us-central1/keyRings/datastream/cryptoKeys/stream-key"

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

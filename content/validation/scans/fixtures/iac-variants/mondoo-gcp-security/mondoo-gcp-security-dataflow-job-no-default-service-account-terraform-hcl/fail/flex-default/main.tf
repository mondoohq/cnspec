# Non-compliant: Flex Template job runs as the default Compute Engine service account.
resource "google_dataflow_flex_template_job" "fail_example" {
  provider                = google-beta
  name                    = "flex-etl-job"
  container_spec_gcs_path = "gs://my-bucket/templates/streaming.json"
  service_account_email   = "987654321098-compute@developer.gserviceaccount.com"
}

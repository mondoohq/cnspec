# Non-compliant: Dataflow Flex Template job has no CMEK.
resource "google_dataflow_flex_template_job" "fail_example" {
  provider                = google-beta
  name                    = "flex-etl-job"
  container_spec_gcs_path = "gs://my-bucket/templates/streaming.json"
}

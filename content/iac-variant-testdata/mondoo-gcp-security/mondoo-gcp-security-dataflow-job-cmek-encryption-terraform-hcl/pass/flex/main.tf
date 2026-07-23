# Compliant: Dataflow Flex Template job encrypts with a customer-managed key.
resource "google_dataflow_flex_template_job" "pass_example" {
  provider                = google-beta
  name                    = "flex-etl-job"
  container_spec_gcs_path = "gs://my-bucket/templates/streaming.json"
  kms_key_name            = "projects/my-project/locations/us-central1/keyRings/dataflow/cryptoKeys/job-key"
}

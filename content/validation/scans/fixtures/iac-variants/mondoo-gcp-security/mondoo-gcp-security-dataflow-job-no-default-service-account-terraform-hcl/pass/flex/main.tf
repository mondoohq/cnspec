# Compliant: Dataflow Flex Template job runs as a dedicated custom service account.
resource "google_dataflow_flex_template_job" "pass_example" {
  provider                = google-beta
  name                    = "flex-etl-job"
  container_spec_gcs_path = "gs://my-bucket/templates/streaming.json"
  service_account_email   = "dataflow-runner@my-project.iam.gserviceaccount.com"
}

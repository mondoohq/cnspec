# Non-compliant: Dataflow job runs as the default Compute Engine service account.
resource "google_dataflow_job" "fail_example" {
  name                  = "etl-job"
  template_gcs_path     = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location     = "gs://my-bucket/tmp"
  service_account_email = "123456789012-compute@developer.gserviceaccount.com"
}

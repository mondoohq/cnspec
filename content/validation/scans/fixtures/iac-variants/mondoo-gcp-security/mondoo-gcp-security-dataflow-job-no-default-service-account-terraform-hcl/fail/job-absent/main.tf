# Non-compliant: Dataflow job sets no service account, defaulting to the Compute Engine SA.
resource "google_dataflow_job" "fail_example" {
  name              = "etl-job"
  template_gcs_path = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location = "gs://my-bucket/tmp"
}

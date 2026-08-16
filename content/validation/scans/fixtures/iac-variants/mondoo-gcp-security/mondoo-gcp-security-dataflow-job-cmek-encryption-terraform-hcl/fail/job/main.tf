# Non-compliant: Dataflow job has no CMEK; uses Google-managed encryption.
resource "google_dataflow_job" "fail_example" {
  name              = "etl-job"
  template_gcs_path = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location = "gs://my-bucket/tmp"
}

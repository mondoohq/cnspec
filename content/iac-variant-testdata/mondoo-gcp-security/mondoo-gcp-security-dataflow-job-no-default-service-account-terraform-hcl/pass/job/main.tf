# Compliant: Dataflow job runs as a dedicated custom service account.
resource "google_dataflow_job" "pass_example" {
  name                  = "etl-job"
  template_gcs_path     = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location     = "gs://my-bucket/tmp"
  service_account_email = "dataflow-runner@my-project.iam.gserviceaccount.com"
}

# Compliant: Dataflow job encrypts with a customer-managed key.
resource "google_dataflow_job" "pass_example" {
  name              = "etl-job"
  template_gcs_path = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location = "gs://my-bucket/tmp"
  kms_key_name      = "projects/my-project/locations/us-central1/keyRings/dataflow/cryptoKeys/job-key"
}

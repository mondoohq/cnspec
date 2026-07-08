# Non-compliant: Dataflow job workers are assigned public IPs.
resource "google_dataflow_job" "fail_example" {
  name              = "etl-job"
  template_gcs_path = "gs://dataflow-templates/latest/Word_Count"
  temp_gcs_location = "gs://my-bucket/tmp"
  ip_configuration  = "WORKER_IP_PUBLIC"
}

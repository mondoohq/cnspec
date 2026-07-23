# Compliant: Flex Template job workers use private IPs only.
resource "google_dataflow_flex_template_job" "pass_example" {
  provider                = google-beta
  name                    = "flex-etl-job"
  container_spec_gcs_path = "gs://my-bucket/templates/streaming.json"
  ip_configuration        = "WORKER_IP_PRIVATE"
}

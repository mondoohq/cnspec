# Compliant: gen1 function routes egress through a VPC connector.
resource "google_cloudfunctions_function" "pass_example" {
  name    = "app-fn"
  runtime = "nodejs20"

  available_memory_mb   = 256
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
  trigger_http          = true
  entry_point           = "handler"

  vpc_connector                 = "projects/my-project/locations/us-central1/connectors/serverless-conn"
  vpc_connector_egress_settings = "ALL_TRAFFIC"
}

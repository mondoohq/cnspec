# Compliant: gen2 function routes egress through a VPC connector.
# In gen2 the connector is set inside the service_config block.
resource "google_cloudfunctions2_function" "pass_example" {
  name     = "app-fn-v2"
  location = "us-central1"

  build_config {
    runtime     = "nodejs20"
    entry_point = "handler"
    source {
      storage_source {
        bucket = "my-bucket"
        object = "index.zip"
      }
    }
  }

  service_config {
    max_instance_count    = 3
    available_memory      = "256M"
    vpc_connector         = "projects/my-project/locations/us-central1/connectors/serverless-conn"
    vpc_connector_egress_settings = "ALL_TRAFFIC"
  }
}

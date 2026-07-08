# Violation: gen2 function has no VPC connector configured.
resource "google_cloudfunctions2_function" "fail_example" {
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
    max_instance_count = 3
    available_memory   = "256M"
  }
}

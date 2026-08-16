# Compliant: gen2 function whose service_config env vars are non-sensitive.
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
    max_instance_count = 3
    available_memory   = "256M"
    environment_variables = {
      LOG_LEVEL   = "info"
      ENVIRONMENT = "production"
    }
  }
}

# Violation: gen2 function service_config stores a plaintext API_KEY env var.
resource "google_cloudfunctions2_function" "fail_example" {
  name     = "leaky-fn-v2"
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
      LOG_LEVEL = "info"
      API_KEY   = "AIzaSyExamplePlaintextKey"
    }
  }
}

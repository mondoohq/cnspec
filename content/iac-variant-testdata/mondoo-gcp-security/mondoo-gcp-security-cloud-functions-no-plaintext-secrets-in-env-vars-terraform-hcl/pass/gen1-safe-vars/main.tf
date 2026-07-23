# Compliant: gen1 function with only non-sensitive env var keys.
resource "google_cloudfunctions_function" "pass_example" {
  name    = "app-fn"
  runtime = "nodejs20"

  available_memory_mb   = 256
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
  trigger_http          = true
  entry_point           = "handler"

  environment_variables = {
    LOG_LEVEL   = "info"
    ENVIRONMENT = "production"
    REGION      = "us-central1"
  }
}

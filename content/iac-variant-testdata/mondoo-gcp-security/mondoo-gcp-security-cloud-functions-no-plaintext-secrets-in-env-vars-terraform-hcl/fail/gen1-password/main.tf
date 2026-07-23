# Violation: gen1 function stores a plaintext DATABASE_PASSWORD env var.
resource "google_cloudfunctions_function" "fail_example" {
  name    = "leaky-fn"
  runtime = "nodejs20"

  available_memory_mb   = 256
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
  trigger_http          = true
  entry_point           = "handler"

  environment_variables = {
    LOG_LEVEL         = "info"
    DATABASE_PASSWORD = "s3cr3t-p4ss"
  }
}

# Violation: gen1 function has no VPC connector configured.
resource "google_cloudfunctions_function" "fail_example" {
  name    = "app-fn"
  runtime = "nodejs20"

  available_memory_mb   = 256
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
  trigger_http          = true
  entry_point           = "handler"
}

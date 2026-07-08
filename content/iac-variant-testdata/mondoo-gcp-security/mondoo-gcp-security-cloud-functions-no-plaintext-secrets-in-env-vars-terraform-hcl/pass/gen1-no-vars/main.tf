# Compliant: gen1 function with no environment_variables at all.
resource "google_cloudfunctions_function" "pass_example" {
  name    = "no-vars-fn"
  runtime = "python312"

  available_memory_mb   = 256
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
  trigger_http          = true
  entry_point           = "main"
}

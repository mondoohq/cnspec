resource "google_cloudfunctions_function" "fail" {
  name                  = "gen1-fn"
  runtime               = "nodejs18"
  region                = "us-central1"
  service_account_email = "my-project@appspot.gserviceaccount.com"

  available_memory_mb   = 256
  entry_point           = "helloGET"
  trigger_http          = true
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
}

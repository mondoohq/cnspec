resource "google_cloudfunctions_function" "pass" {
  name         = "gen1-fn"
  runtime      = "nodejs18"
  region       = "us-central1"
  kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"

  available_memory_mb   = 256
  entry_point           = "helloGET"
  trigger_http          = true
  source_archive_bucket = "my-bucket"
  source_archive_object = "index.zip"
}

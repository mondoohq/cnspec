resource "google_cloudfunctions2_function" "pass" {
  name         = "gen2-fn"
  location     = "us-central1"
  kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"

  build_config {
    runtime     = "nodejs18"
    entry_point = "helloGET"
    source {
      storage_source {
        bucket = "my-bucket"
        object = "index.zip"
      }
    }
  }

  service_config {
    max_instance_count = 1
    available_memory   = "256M"
  }
}

resource "google_cloudfunctions2_function" "pass" {
  name             = "gen2-fn"
  location         = "us-central1"
  ingress_settings = "ALLOW_INTERNAL_ONLY"

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
    ingress_settings   = "ALLOW_INTERNAL_ONLY"
  }
}

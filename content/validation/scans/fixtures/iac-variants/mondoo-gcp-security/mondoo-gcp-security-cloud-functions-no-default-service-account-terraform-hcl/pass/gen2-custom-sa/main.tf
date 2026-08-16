resource "google_cloudfunctions2_function" "pass" {
  name                  = "gen2-fn"
  location              = "us-central1"
  service_account_email = "custom-fn-sa@my-project.iam.gserviceaccount.com"

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
    max_instance_count    = 1
    available_memory      = "256M"
    service_account_email = "custom-fn-sa@my-project.iam.gserviceaccount.com"
  }
}

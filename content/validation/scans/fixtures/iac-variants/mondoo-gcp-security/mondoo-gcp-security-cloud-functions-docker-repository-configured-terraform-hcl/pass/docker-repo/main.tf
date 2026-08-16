resource "google_cloudfunctions2_function" "pass" {
  name     = "gen2-fn"
  location = "us-central1"

  build_config {
    runtime           = "nodejs18"
    entry_point       = "helloGET"
    docker_repository = "projects/my-project/locations/us-central1/repositories/my-repo"
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

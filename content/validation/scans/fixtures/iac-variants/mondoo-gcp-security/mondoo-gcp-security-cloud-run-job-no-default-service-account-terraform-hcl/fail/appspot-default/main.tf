# Non-compliant: uses the default App Engine service account.
resource "google_cloud_run_v2_job" "job" {
  name     = "appspot-sa-job"
  location = "us-central1"

  template {
    template {
      service_account = "my-project@appspot.gserviceaccount.com"

      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
